package audio

// Buffer is a chunk of planar PCM.
type Buffer struct {
	Fmt Format

	I []int32
	F []float32

	Stride int
	N      int

	Pos     int64
	Discont bool
}

// Cap returns the buffer's frame capacity per channel.
func (b *Buffer) Cap() int { return b.Stride }

// ChanI returns channel c's valid frames as a contiguous int32 slice.
func (b *Buffer) ChanI(c int) []int32 {
	if b.I == nil {
		panic("audio: ChanI on a float-domain buffer")
	}
	return b.I[c*b.Stride : c*b.Stride+b.N : c*b.Stride+b.Stride]
}

// ChanF returns channel c's valid frames as a contiguous float32 slice.
func (b *Buffer) ChanF(c int) []float32 {
	if b.F == nil {
		panic("audio: ChanF on an int-domain buffer")
	}
	return b.F[c*b.Stride : c*b.Stride+b.N : c*b.Stride+b.Stride]
}

// CopyFrames copies n frames from src starting at frame srcOff into dst starting at frame dstOff, channel by channel.
func CopyFrames(dst *Buffer, dstOff int, src *Buffer, srcOff, n int) {
	if n <= 0 {
		return
	}
	if dst.Fmt != src.Fmt {
		panic("audio: CopyFrames between mismatched formats " + dst.Fmt.String() + " and " + src.Fmt.String())
	}
	for c := 0; c < dst.Fmt.Channels; c++ {
		if dst.Fmt.Type == Float {
			copy(dst.F[c*dst.Stride+dstOff:c*dst.Stride+dstOff+n],
				src.F[c*src.Stride+srcOff:c*src.Stride+srcOff+n])
		} else {
			copy(dst.I[c*dst.Stride+dstOff:c*dst.Stride+dstOff+n],
				src.I[c*src.Stride+srcOff:c*src.Stride+srcOff+n])
		}
	}
}
