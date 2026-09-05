package dsp

import (
	"fmt"
	"io"
	"time"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/dsp/dither"
	"novelhub/pkg/waxflow/dsp/gain"
	"novelhub/pkg/waxflow/dsp/mix"
	"novelhub/pkg/waxflow/dsp/resample"
	"novelhub/pkg/waxflow/waxerr"
)

type scratchBox struct{ buf *audio.Buffer }

func (b *scratchBox) get(f audio.Format, frames int) *audio.Buffer {
	if b.buf == nil || b.buf.Fmt != f || b.buf.Cap() != frames {
		audio.Put(b.buf)
		b.buf = audio.Get(f, frames)
	}
	return b.buf
}

func (b *scratchBox) release() {
	audio.Put(b.buf)
	b.buf = nil
}

func pull1to1(up Stage, box *scratchBox, dst *audio.Buffer) (*audio.Buffer, error) {
	dst.N = 0
	in := box.get(up.Format(), dst.Cap())
	if err := up.ReadChunk(in); err != nil {
		return nil, err
	}
	dst.N = in.N
	dst.Pos = in.Pos
	dst.Discont = in.Discont
	return in, nil
}

type convertStage struct {
	up    Stage
	fmt   audio.Format
	scale float32
	box   scratchBox
}

func (s *convertStage) Format() audio.Format { return s.fmt }
func (s *convertStage) release()             { s.box.release() }

func (s *convertStage) ReadChunk(dst *audio.Buffer) error {
	in, err := pull1to1(s.up, &s.box, dst)
	if err != nil {
		return err
	}
	for c := 0; c < s.fmt.Channels; c++ {
		src := in.ChanI(c)
		out := dst.ChanF(c)
		for i := range src {
			out[i] = float32(src[i]) * s.scale
		}
	}
	return nil
}

type widenStage struct {
	up    Stage
	fmt   audio.Format
	shift uint
	box   scratchBox
}

func (s *widenStage) Format() audio.Format { return s.fmt }
func (s *widenStage) release()             { s.box.release() }

func (s *widenStage) ReadChunk(dst *audio.Buffer) error {
	in, err := pull1to1(s.up, &s.box, dst)
	if err != nil {
		return err
	}
	for c := 0; c < s.fmt.Channels; c++ {
		src := in.ChanI(c)
		out := dst.ChanI(c)
		for i := range src {
			out[i] = src[i] << s.shift
		}
	}
	return nil
}

type mixStage struct {
	up     Stage
	fmt    audio.Format
	matrix *mix.Matrix
	box    scratchBox
	srcV   [][]float32
	dstV   [][]float32
}

func (s *mixStage) Format() audio.Format { return s.fmt }
func (s *mixStage) release()             { s.box.release() }

func (s *mixStage) ReadChunk(dst *audio.Buffer) error {
	in, err := pull1to1(s.up, &s.box, dst)
	if err != nil {
		return err
	}
	if s.srcV == nil {
		s.srcV = make([][]float32, s.matrix.In())
		s.dstV = make([][]float32, s.matrix.Out())
	}
	for c := range s.srcV {
		s.srcV[c] = in.ChanF(c)
	}
	for c := range s.dstV {
		s.dstV[c] = dst.ChanF(c)
	}
	s.matrix.Apply(s.dstV, s.srcV, in.N)
	return nil
}

type gainStage struct {
	up  Stage
	fmt audio.Format
	g   float32
}

func (s *gainStage) Format() audio.Format { return s.fmt }

func (s *gainStage) ReadChunk(dst *audio.Buffer) error {
	if err := s.up.ReadChunk(dst); err != nil {
		return err
	}
	for c := 0; c < s.fmt.Channels; c++ {
		gain.Apply(dst.ChanF(c), s.g)
	}
	return nil
}

type quantizeStage struct {
	up  Stage
	fmt audio.Format
	q   *dither.Quantizer
	box scratchBox
}

func (s *quantizeStage) Format() audio.Format { return s.fmt }
func (s *quantizeStage) release()             { s.box.release() }

func (s *quantizeStage) ReadChunk(dst *audio.Buffer) error {
	in, err := pull1to1(s.up, &s.box, dst)
	if err != nil {
		return err
	}
	if in.Discont {
		s.q.Reset()
	}
	for c := 0; c < s.fmt.Channels; c++ {
		s.q.Quantize(dst.ChanI(c), in.ChanF(c), c, in.Pos)
	}
	return nil
}

type kernelOps interface {
	process(dst, src [][]float32) (produced, consumed int)
	drain(dst [][]float32) int
	anchor(pos int64) int64
}

type resampleOps struct{ k *resample.Resampler }

func (o resampleOps) process(dst, src [][]float32) (int, int) { return o.k.Process(dst, src) }
func (o resampleOps) drain(dst [][]float32) int               { return o.k.Drain(dst) }
func (o resampleOps) anchor(pos int64) int64 {
	outPos, phase := o.k.OffsetFor(pos)
	o.k.Reset(phase)
	return outPos
}

type limiterOps struct{ k *gain.Limiter }

func (o limiterOps) process(dst, src [][]float32) (int, int) { return o.k.Process(dst, src) }
func (o limiterOps) drain(dst [][]float32) int               { return o.k.Drain(dst) }
func (o limiterOps) anchor(pos int64) int64 {
	o.k.Reset()
	return pos
}
func (o limiterOps) Horizon() time.Duration { return o.k.Horizon() }

type compressorOps struct{ k *gain.Compressor }

func (o compressorOps) process(dst, src [][]float32) (int, int) { return o.k.Process(dst, src) }
func (o compressorOps) drain(dst [][]float32) int               { return o.k.Drain(dst) }
func (o compressorOps) anchor(pos int64) int64 {
	o.k.Reset()
	return pos
}
func (o compressorOps) Horizon() time.Duration { return o.k.Horizon() }

type pumpStage struct {
	up    Stage
	fmt   audio.Format
	inFmt audio.Format
	ops   kernelOps

	box  scratchBox
	off  int
	srcV [][]float32
	dstV [][]float32

	outPos      int64
	anchorPos   int64
	needAnchor  bool
	splice      bool
	markDiscont bool
	started     bool
	eof         bool
}

func newPump(up Stage, out audio.Format, ops kernelOps) *pumpStage {
	return &pumpStage{up: up, fmt: out, inFmt: up.Format(), ops: ops}
}

func (s *pumpStage) Format() audio.Format { return s.fmt }
func (s *pumpStage) release()             { s.box.release() }

func (s *pumpStage) horizon() time.Duration {
	if st, ok := s.ops.(Settler); ok {
		return st.Horizon()
	}
	return 0
}

func (s *pumpStage) ReadChunk(dst *audio.Buffer) error {
	if s.srcV == nil {
		s.srcV = make([][]float32, s.inFmt.Channels)
		s.dstV = make([][]float32, s.fmt.Channels)
	}
	dst.N = 0
	produced := 0
	var chunkPos int64
	discont := false

	mark := func(p int) {
		if p > 0 && produced == 0 {
			chunkPos = s.outPos
			discont = s.markDiscont
			s.markDiscont = false
		}
		produced += p
		s.outPos += int64(p)
	}

pump:
	for produced < dst.Cap() {
		if s.needAnchor {
			s.outPos = s.ops.anchor(s.anchorPos)
			s.markDiscont = true
			s.needAnchor = false
		}
		for c := range s.dstV {
			s.dstV[c] = dst.F[c*dst.Stride+produced : c*dst.Stride+dst.Cap()]
		}

		switch in := s.box.buf; {
		case s.splice:
			if p := s.ops.drain(s.dstV); p > 0 {
				mark(p)
				break
			}
			s.splice = false
			if produced > 0 {
				s.needAnchor = true
				break pump
			}
			s.outPos = s.ops.anchor(s.anchorPos)
			s.markDiscont = true

		case in != nil && s.off < in.N:
			for c := range s.srcV {
				s.srcV[c] = in.F[c*in.Stride+s.off : c*in.Stride+in.N]
			}
			p, consumed := s.ops.process(s.dstV, s.srcV)
			mark(p)
			s.off += consumed

		case s.eof:
			p := s.ops.drain(s.dstV)
			if p == 0 {
				break pump
			}
			mark(p)

		default:
			in := s.box.get(s.inFmt, dst.Cap())
			err := s.up.ReadChunk(in)
			if err == io.EOF {
				s.eof = true
				continue
			}
			if err != nil {
				return err
			}
			s.off = 0
			switch {
			case !s.started:
				s.started = true
				s.outPos = s.ops.anchor(in.Pos)
				s.markDiscont = in.Discont
			case in.Discont:
				s.anchorPos = in.Pos
				s.splice = true
			}
		}
	}
	if produced == 0 {
		return io.EOF
	}
	dst.N = produced
	dst.Pos = chunkPos
	dst.Discont = discont
	return nil
}

type framerStage struct {
	up   Stage
	fmt  audio.Format
	size int

	box         scratchBox
	off         int
	pendDiscont bool
	eof         bool
}

func (s *framerStage) Format() audio.Format { return s.fmt }
func (s *framerStage) release()             { s.box.release() }

func (s *framerStage) ReadChunk(dst *audio.Buffer) error {
	if dst.Cap() < s.size {
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("dsp: framer emits %d-frame chunks, buffer holds %d", s.size, dst.Cap()))
	}
	dst.N = 0
	for dst.N < s.size {
		in := s.box.buf
		if in == nil || s.off >= in.N {
			if s.eof {
				break
			}
			in = s.box.get(s.fmt, max(s.size, audio.StandardChunk))
			err := s.up.ReadChunk(in)
			if err == io.EOF {
				s.eof = true
				in.N = 0
				break
			}
			if err != nil {
				return err
			}
			s.off = 0
			if in.Discont {
				s.pendDiscont = true
				if dst.N > 0 {
					break
				}
			}
		}
		if dst.N == 0 {
			dst.Pos = in.Pos + int64(s.off)
			dst.Discont = s.pendDiscont
			s.pendDiscont = false
		}
		n := min(s.size-dst.N, in.N-s.off)
		audio.CopyFrames(dst, dst.N, in, s.off, n)
		dst.N += n
		s.off += n
	}
	if dst.N == 0 {
		return io.EOF
	}
	return nil
}
