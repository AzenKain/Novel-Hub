package alac

import (
	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/waxerr"
)

var (
	_ codec.Decoder  = (*Decoder)(nil)
	_ codec.Releaser = (*Decoder)(nil)
)

const (
	idSCE = 0
	idCPE = 1
	idCCE = 2
	idLFE = 3
	idDSE = 4
	idPCE = 5
	idFIL = 6
	idEND = 7
)

// Decoder decodes ALAC packets into planar int buffers.
type Decoder struct {
	ac  Config
	fmt audio.Format
	buf *audio.Buffer

	predictor []int32
	mixU      []int32
	mixV      []int32
	shiftBuf  []uint16
	coefsU    [32]int16
	coefsV    [32]int16
}

// NewDecoder returns a Decoder for a stream.
func NewDecoder(cfg Config, f audio.Format) (*Decoder, error) {
	if err := f.Valid(); err != nil {
		return nil, err
	}
	if want := cfg.Format(); f != want {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			"alac: track format "+f.String()+" does not match the magic cookie ("+want.String()+")")
	}
	n := int(cfg.FrameLength)
	return &Decoder{
		ac:        cfg,
		fmt:       f,
		predictor: make([]int32, n),
		mixU:      make([]int32, n),
		mixV:      make([]int32, n),
		shiftBuf:  make([]uint16, 2*n),
	}, nil
}

// Decode decodes one packet and emits one buffer.
func (d *Decoder) Decode(pkt []byte, emit func(*audio.Buffer) error) error {
	frameLen := int(d.ac.FrameLength)
	if d.buf == nil || d.buf.Cap() < frameLen || d.buf.Fmt != d.fmt {
		audio.Put(d.buf)
		d.buf = audio.Get(d.fmt, frameLen)
	}
	r := &bitReader{data: pkt, pos: 0, validBits: len(pkt) * 8}

	numChannels := d.ac.Channels
	channelIndex := 0
	frameSamples := 0
	for channelIndex < numChannels {
		if r.overrun() {
			return malformed("packet ends mid-frame")
		}
		tag := r.read(3)
		switch tag {
		case idSCE, idLFE:
			n, err := d.decodeMono(r, channelIndex)
			if err != nil {
				return err
			}
			frameSamples = n
			channelIndex++
		case idCPE:
			if channelIndex+2 > numChannels {
				channelIndex = numChannels
				continue
			}
			n, err := d.decodeStereo(r, channelIndex)
			if err != nil {
				return err
			}
			frameSamples = n
			channelIndex += 2
		case idDSE:
			if err := skipDSE(r); err != nil {
				return err
			}
		case idFIL:
			if err := skipFIL(r); err != nil {
				return err
			}
		case idEND:
			r.byteAlign()
			channelIndex = numChannels
		default:
			return malformed("unsupported element type %d", tag)
		}
	}
	if r.overrun() {
		return malformed("frame overruns packet")
	}
	for c := channelIndex; c < numChannels; c++ {
		clear(d.buf.I[c*d.buf.Stride : c*d.buf.Stride+frameSamples])
	}
	d.buf.N = frameSamples
	return emit(d.buf)
}

func (d *Decoder) elementHeader(r *bitReader, sideBit uint) (numSamples int, bytesShifted uint32, escape uint32, chanBits uint, err error) {
	r.read(4)
	if r.read(12) != 0 {
		return 0, 0, 0, 0, malformed("element header reserved bits set")
	}
	hdr := r.read(4)
	partial := hdr >> 3
	bytesShifted = (hdr >> 1) & 0x3
	if bytesShifted == 3 {
		return 0, 0, 0, 0, malformed("invalid shift-off value")
	}
	escape = hdr & 1
	chanBits = uint(d.ac.BitDepth) - uint(bytesShifted)*8 + sideBit
	numSamples = int(d.ac.FrameLength)
	if partial != 0 {
		numSamples = int(r.read(16)<<16 | r.read(16))
	}
	if numSamples <= 0 || numSamples > int(d.ac.FrameLength) {
		return 0, 0, 0, 0, malformed("partial frame length %d outside 1..%d", numSamples, d.ac.FrameLength)
	}
	return numSamples, bytesShifted, escape, chanBits, nil
}

func (d *Decoder) decodeMono(r *bitReader, ci int) (int, error) {
	numSamples, bytesShifted, escape, chanBits, err := d.elementHeader(r, 0)
	if err != nil {
		return 0, err
	}
	shift := uint(bytesShifted * 8)
	if escape == 0 {
		r.read(8)
		r.read(8)
		hb := r.read(8)
		modeU, denShiftU := hb>>4, uint(hb&0xf)
		hb = r.read(8)
		pbFactorU, numU := hb>>5, int(hb&0x1f)
		for i := 0; i < numU; i++ {
			d.coefsU[i] = int16(r.read(16))
		}
		shiftStart := 0
		if bytesShifted != 0 {
			shiftStart = r.pos
			r.advance(int(shift) * numSamples)
		}
		if err := d.dynDecomp(r, d.predictor[:numSamples], chanBits, (d.ac.PB*pbFactorU)/4); err != nil {
			return 0, err
		}
		d.predict(modeU, numSamples, d.coefsU[:numU], numU, chanBits, denShiftU, d.mixU)
		if bytesShifted != 0 {
			d.readShift(r, shiftStart, shift, numSamples, 1)
		}
	} else {
		d.readUncompressed(r, chanBits, numSamples, d.mixU, nil)
		bytesShifted = 0
	}
	dst := d.buf.I[ci*d.buf.Stride : ci*d.buf.Stride+numSamples]
	for i := 0; i < numSamples; i++ {
		v := d.mixU[i]
		if bytesShifted != 0 {
			v = v<<shift | int32(d.shiftBuf[i])
		}
		dst[i] = v
	}
	return numSamples, nil
}

func (d *Decoder) decodeStereo(r *bitReader, ci int) (int, error) {
	numSamples, bytesShifted, escape, chanBits, err := d.elementHeader(r, 1)
	if err != nil {
		return 0, err
	}
	shift := uint(bytesShifted * 8)
	var mixBits, mixRes int32
	if escape == 0 {
		mixBits = int32(r.read(8))
		mixRes = int32(int8(r.read(8)))
		if mixRes != 0 && mixBits >= 32 {
			return 0, malformed("mix shift %d out of range", mixBits)
		}
		hb := r.read(8)
		modeU, denShiftU := hb>>4, uint(hb&0xf)
		hb = r.read(8)
		pbFactorU, numU := hb>>5, int(hb&0x1f)
		for i := 0; i < numU; i++ {
			d.coefsU[i] = int16(r.read(16))
		}
		hb = r.read(8)
		modeV, denShiftV := hb>>4, uint(hb&0xf)
		hb = r.read(8)
		pbFactorV, numV := hb>>5, int(hb&0x1f)
		for i := 0; i < numV; i++ {
			d.coefsV[i] = int16(r.read(16))
		}
		shiftStart := 0
		if bytesShifted != 0 {
			shiftStart = r.pos
			r.advance(int(shift) * 2 * numSamples)
		}
		if err := d.dynDecomp(r, d.predictor[:numSamples], chanBits, (d.ac.PB*pbFactorU)/4); err != nil {
			return 0, err
		}
		d.predict(modeU, numSamples, d.coefsU[:numU], numU, chanBits, denShiftU, d.mixU)
		if err := d.dynDecomp(r, d.predictor[:numSamples], chanBits, (d.ac.PB*pbFactorV)/4); err != nil {
			return 0, err
		}
		d.predict(modeV, numSamples, d.coefsV[:numV], numV, chanBits, denShiftV, d.mixV)
		if bytesShifted != 0 {
			d.readShift(r, shiftStart, shift, numSamples, 2)
		}
	} else {
		chanBits = uint(d.ac.BitDepth)
		d.readUncompressed(r, chanBits, numSamples, d.mixU, d.mixV)
		mixBits, mixRes = 0, 0
		bytesShifted = 0
	}
	stride := d.buf.Stride
	dl := d.buf.I[ci*stride : ci*stride+numSamples]
	dr := d.buf.I[(ci+1)*stride : (ci+1)*stride+numSamples]
	for i := 0; i < numSamples; i++ {
		var l, r0 int32
		if mixRes != 0 {
			l = d.mixU[i] + d.mixV[i] - ((mixRes * d.mixV[i]) >> uint(mixBits))
			r0 = l - d.mixV[i]
		} else {
			l, r0 = d.mixU[i], d.mixV[i]
		}
		if bytesShifted != 0 {
			l = l<<shift | int32(d.shiftBuf[2*i])
			r0 = r0<<shift | int32(d.shiftBuf[2*i+1])
		}
		dl[i], dr[i] = l, r0
	}
	return numSamples, nil
}

func (d *Decoder) predict(mode uint32, num int, coefs []int16, numactive int, chanBits, denShift uint, out []int32) {
	if mode == 0 {
		unpcBlock(d.predictor, out, num, coefs, numactive, chanBits, denShift)
		return
	}
	unpcBlock(d.predictor, d.predictor, num, nil, 31, chanBits, 0)
	unpcBlock(d.predictor, out, num, coefs, numactive, chanBits, denShift)
}

func (d *Decoder) readUncompressed(r *bitReader, chanBits uint, num int, u, v []int32) {
	sh := 32 - chanBits
	for i := 0; i < num; i++ {
		u[i] = int32(r.read(chanBits)) << sh >> sh
		if v != nil {
			v[i] = int32(r.read(chanBits)) << sh >> sh
		}
	}
}

func (d *Decoder) readShift(r *bitReader, start int, shift uint, num, count int) {
	sr := bitReader{data: r.data, pos: start, validBits: r.validBits}
	for i := 0; i < num*count; i++ {
		d.shiftBuf[i] = uint16(sr.read(shift))
	}
}

func (d *Decoder) dynDecomp(r *bitReader, pc []int32, chanBits uint, pb uint32) error {
	numSamples := len(pc)
	maxSize := uint32(chanBits)
	mb := d.ac.MB
	kb := d.ac.KB
	wb := uint32(1)<<kb - 1
	zmode := uint32(0)
	p := r.pos
	c := 0
	for c < numSamples {
		if p > r.validBits {
			return malformed("adaptive-Golomb data overruns packet")
		}
		m := mb >> qbShift
		k := lg3a(m)
		if k > kb {
			k = kb
		}
		m = 1<<k - 1
		n := r.dynGet32(&p, m, k, maxSize)

		ndecode := n + zmode
		del := int32((ndecode + 1) >> 1)
		if ndecode&1 != 0 {
			del = -del
		}
		pc[c] = del
		c++

		mb = pb*(n+zmode) + mb - ((pb * mb) >> qbShift)
		if n > nMaxMeanClamp {
			mb = nMeanClampVal
		}
		zmode = 0

		if (mb<<mmulShift) < qb && c < numSamples {
			zmode = 1
			k = uint32(int(lead(mb)) - bitOff + int((mb+mOff)>>mDenShift))
			mz := (uint32(1)<<k - 1) & wb
			n = r.dynGet16(&p, mz, k)
			if int64(c)+int64(n) > int64(numSamples) {
				return malformed("zero run overruns frame")
			}
			for j := uint32(0); j < n; j++ {
				pc[c] = 0
				c++
			}
			if n >= 65535 {
				zmode = 0
			}
			mb = 0
		}
	}
	r.pos = p
	return nil
}

func signOf(i int32) int32 {
	switch {
	case i > 0:
		return 1
	case i < 0:
		return -1
	default:
		return 0
	}
}

func unpcBlock(pc1, out []int32, num int, coefs []int16, numactive int, chanBits, denShift uint) {
	if num <= 0 {
		return
	}
	var chanShift uint
	if chanBits < 32 {
		chanShift = 32 - chanBits
	}
	var denHalf int32
	if denShift > 0 {
		denHalf = int32(1) << (denShift - 1)
	}

	out[0] = pc1[0]
	if numactive == 0 {
		copy(out[1:num], pc1[1:num])
		return
	}
	if numactive == 31 {
		prev := out[0]
		for j := 1; j < num; j++ {
			prev = (pc1[j] + prev) << chanShift >> chanShift
			out[j] = prev
		}
		return
	}
	warm := numactive
	if warm > num-1 {
		warm = num - 1
	}
	for j := 1; j <= warm; j++ {
		out[j] = (pc1[j] + out[j-1]) << chanShift >> chanShift
	}
	lim := numactive + 1
	for j := lim; j < num; j++ {
		top := out[j-lim]
		var sum1 int32
		for k := 0; k < numactive; k++ {
			sum1 += int32(coefs[k]) * (out[j-1-k] - top)
		}
		del := pc1[j]
		del0 := del
		sg := signOf(del)
		del += top + ((sum1 + denHalf) >> denShift)
		out[j] = del << chanShift >> chanShift

		if sg > 0 {
			for k := numactive - 1; k >= 0; k-- {
				dd := top - out[j-1-k]
				sgn := signOf(dd)
				coefs[k] -= int16(sgn)
				del0 -= int32(numactive-k) * ((sgn * dd) >> denShift)
				if del0 <= 0 {
					break
				}
			}
		} else if sg < 0 {
			for k := numactive - 1; k >= 0; k-- {
				dd := top - out[j-1-k]
				sgn := signOf(dd)
				coefs[k] += int16(sgn)
				del0 -= int32(numactive-k) * ((-sgn * dd) >> denShift)
				if del0 >= 0 {
					break
				}
			}
		}
	}
}

func skipDSE(r *bitReader) error {
	r.read(4)
	align := r.one()
	count := r.read(8)
	if count == 255 {
		count += r.read(8)
	}
	if align != 0 {
		r.byteAlign()
	}
	r.advance(int(count) * 8)
	return nil
}

func skipFIL(r *bitReader) error {
	count := r.read(4)
	if count == 15 {
		count += r.read(8) - 1
	}
	r.advance(int(count) * 8)
	return nil
}

// Drain is a no-op: ALAC has no decoder latency.
func (d *Decoder) Drain(func(*audio.Buffer) error) error { return nil }

// Reset is a no-op: every ALAC frame decodes independently.
func (d *Decoder) Reset() {}

// Release returns the output buffer to the pool (codec.Releaser).
func (d *Decoder) Release() {
	audio.Put(d.buf)
	d.buf = nil
}
