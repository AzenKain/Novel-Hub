// Package fft provides the shared forward complex FFT kernel the codec transforms build on (the CELT MDCT today; any future MDCT/DFT consumer).
package fft

import (
	"fmt"
	"math"
)

// Plan holds the read-only permutation and twiddle tables for one transform length.
type Plan struct {
	n      int
	perm   []int32
	levels []level
}

type level struct {
	f, m, bl, nBlocks int
	twRe, twIm        [][]float32
}

// NewPlan builds a Plan for length n, which must be a positive product of 2s, 3s, and 5s.
func NewPlan(n int) *Plan {
	if n < 1 {
		panic(fmt.Sprintf("fft: length %d must be positive", n))
	}
	if n == 1 {
		return &Plan{n: 1, perm: []int32{0}}
	}
	factors := factorize(n)
	if factors == nil {
		panic(fmt.Sprintf("fft: length %d does not factor into 2, 3, and 5", n))
	}
	p := &Plan{n: n}
	p.perm = buildPerm(n, factors)

	bl := n
	stride := 1
	for _, f := range factors {
		m := bl / f
		lv := level{f: f, m: m, bl: bl, nBlocks: stride}
		lv.twRe = make([][]float32, f-1)
		lv.twIm = make([][]float32, f-1)
		for q := 1; q < f; q++ {
			re := make([]float32, m)
			im := make([]float32, m)
			for k2 := 0; k2 < m; k2++ {
				s, c := math.Sincos(-2 * math.Pi * float64((k2*q*stride)%n) / float64(n))
				re[k2] = float32(c)
				im[k2] = float32(s)
			}
			lv.twRe[q-1] = re
			lv.twIm[q-1] = im
		}
		p.levels = append([]level{lv}, p.levels...)
		bl = m
		stride *= f
	}
	return p
}

// N returns the transform length.
func (p *Plan) N() int { return p.n }

// Transform computes the forward DFT of (srcRe, srcIm) into (dstRe, dstIm).
func (p *Plan) Transform(dstRe, dstIm, srcRe, srcIm []float32) {
	if len(dstRe) < p.n || len(dstIm) < p.n || len(srcRe) < p.n || len(srcIm) < p.n {
		panic("fft: buffer shorter than the transform length")
	}
	if &dstRe[0] == &srcRe[0] || &dstRe[0] == &srcIm[0] ||
		&dstIm[0] == &srcRe[0] || &dstIm[0] == &srcIm[0] {
		panic("fft: dst and src must not overlap; the transform is not in-place")
	}
	dstRe, dstIm = dstRe[:p.n], dstIm[:p.n]
	for i, pi := range p.perm {
		dstRe[i] = srcRe[pi]
		dstIm[i] = srcIm[pi]
	}
	for i := range p.levels {
		combineLevel(&p.levels[i], dstRe, dstIm)
	}
}

func factorize(n int) []int {
	var fs []int
	rem := n
	for _, f := range []int{5, 3, 4, 2} {
		for rem%f == 0 {
			fs = append(fs, f)
			rem /= f
		}
	}
	if rem != 1 {
		return nil
	}
	return fs
}

func buildPerm(n int, factors []int) []int32 {
	perm := make([]int32, n)
	lengths := make([]int, len(factors))
	l := n
	for i, f := range factors {
		l /= f
		lengths[i] = l
	}
	var rec func(outBase, inBase, stride, depth int)
	rec = func(outBase, inBase, stride, depth int) {
		f := factors[depth]
		m := lengths[depth]
		if m == 1 {
			for q := 0; q < f; q++ {
				perm[outBase+q] = int32(inBase + q*stride)
			}
			return
		}
		for q := 0; q < f; q++ {
			rec(outBase+q*m, inBase+q*stride, stride*f, depth+1)
		}
	}
	rec(0, 0, 1, 0)
	return perm
}

const (
	fftCos3  = -0.5
	fftSin3  = 0.8660254037844386
	fftCos5a = 0.30901699437494745
	fftCos5b = -0.8090169943749475
	fftSin5a = 0.9510565162951535
	fftSin5b = 0.5877852522924731
)

func combineLevel(lv *level, re, im []float32) {
	switch lv.f {
	case 2:
		combine2(lv, re, im)
	case 3:
		combine3(lv, re, im)
	case 4:
		combine4(lv, re, im)
	case 5:
		combine5(lv, re, im)
	default:
		panic("fft: unsupported radix")
	}
}

func combine2(lv *level, re, im []float32) {
	m := lv.m
	w0r, w0i := lv.twRe[0], lv.twIm[0]
	for base := 0; base < len(re); base += lv.bl {
		for k2 := 0; k2 < m; k2++ {
			i0, i1 := base+k2, base+m+k2
			t1r := re[i1]*w0r[k2] - im[i1]*w0i[k2]
			t1i := re[i1]*w0i[k2] + im[i1]*w0r[k2]
			re[i1] = re[i0] - t1r
			im[i1] = im[i0] - t1i
			re[i0] += t1r
			im[i0] += t1i
		}
	}
}

func combine3(lv *level, re, im []float32) {
	m := lv.m
	w0r, w0i := lv.twRe[0], lv.twIm[0]
	w1r, w1i := lv.twRe[1], lv.twIm[1]
	for base := 0; base < len(re); base += lv.bl {
		for k2 := 0; k2 < m; k2++ {
			i0, i1, i2 := base+k2, base+m+k2, base+2*m+k2
			t1r := re[i1]*w0r[k2] - im[i1]*w0i[k2]
			t1i := re[i1]*w0i[k2] + im[i1]*w0r[k2]
			t2r := re[i2]*w1r[k2] - im[i2]*w1i[k2]
			t2i := re[i2]*w1i[k2] + im[i2]*w1r[k2]
			er, ei := t1r+t2r, t1i+t2i
			or, oi := t1r-t2r, t1i-t2i
			mr := re[i0] + fftCos3*er
			mi := im[i0] + fftCos3*ei
			re[i0] += er
			im[i0] += ei
			re[i1] = mr + fftSin3*oi
			im[i1] = mi - fftSin3*or
			re[i2] = mr - fftSin3*oi
			im[i2] = mi + fftSin3*or
		}
	}
}

func combine4(lv *level, re, im []float32) {
	m := lv.m
	w0r, w0i := lv.twRe[0], lv.twIm[0]
	w1r, w1i := lv.twRe[1], lv.twIm[1]
	w2r, w2i := lv.twRe[2], lv.twIm[2]
	for base := 0; base < len(re); base += lv.bl {
		for k2 := 0; k2 < m; k2++ {
			i0, i1, i2, i3 := base+k2, base+m+k2, base+2*m+k2, base+3*m+k2
			t1r := re[i1]*w0r[k2] - im[i1]*w0i[k2]
			t1i := re[i1]*w0i[k2] + im[i1]*w0r[k2]
			t2r := re[i2]*w1r[k2] - im[i2]*w1i[k2]
			t2i := re[i2]*w1i[k2] + im[i2]*w1r[k2]
			t3r := re[i3]*w2r[k2] - im[i3]*w2i[k2]
			t3i := re[i3]*w2i[k2] + im[i3]*w2r[k2]
			ar, ai := re[i0]+t2r, im[i0]+t2i
			br, bi := re[i0]-t2r, im[i0]-t2i
			cr, ci := t1r+t3r, t1i+t3i
			dr, di := t1r-t3r, t1i-t3i
			re[i0] = ar + cr
			im[i0] = ai + ci
			re[i1] = br + di
			im[i1] = bi - dr
			re[i2] = ar - cr
			im[i2] = ai - ci
			re[i3] = br - di
			im[i3] = bi + dr
		}
	}
}

func combine5(lv *level, re, im []float32) {
	m := lv.m
	w0r, w0i := lv.twRe[0], lv.twIm[0]
	w1r, w1i := lv.twRe[1], lv.twIm[1]
	w2r, w2i := lv.twRe[2], lv.twIm[2]
	w3r, w3i := lv.twRe[3], lv.twIm[3]
	for base := 0; base < len(re); base += lv.bl {
		for k2 := 0; k2 < m; k2++ {
			i0 := base + k2
			i1, i2 := i0+m, i0+2*m
			i3, i4 := i0+3*m, i0+4*m
			t1r := re[i1]*w0r[k2] - im[i1]*w0i[k2]
			t1i := re[i1]*w0i[k2] + im[i1]*w0r[k2]
			t2r := re[i2]*w1r[k2] - im[i2]*w1i[k2]
			t2i := re[i2]*w1i[k2] + im[i2]*w1r[k2]
			t3r := re[i3]*w2r[k2] - im[i3]*w2i[k2]
			t3i := re[i3]*w2i[k2] + im[i3]*w2r[k2]
			t4r := re[i4]*w3r[k2] - im[i4]*w3i[k2]
			t4i := re[i4]*w3i[k2] + im[i4]*w3r[k2]
			e1r, e1i := t1r+t4r, t1i+t4i
			e2r, e2i := t2r+t3r, t2i+t3i
			o1r, o1i := t1r-t4r, t1i-t4i
			o2r, o2i := t2r-t3r, t2i-t3i
			x0r, x0i := re[i0], im[i0]
			re[i0] = x0r + (e1r + e2r)
			im[i0] = x0i + (e1i + e2i)
			a1r := x0r + fftCos5a*e1r + fftCos5b*e2r
			a1i := x0i + fftCos5a*e1i + fftCos5b*e2i
			b1r := fftSin5a*o1r + fftSin5b*o2r
			b1i := fftSin5a*o1i + fftSin5b*o2i
			a2r := x0r + fftCos5b*e1r + fftCos5a*e2r
			a2i := x0i + fftCos5b*e1i + fftCos5a*e2i
			b2r := fftSin5b*o1r - fftSin5a*o2r
			b2i := fftSin5b*o1i - fftSin5a*o2i
			re[i1] = a1r + b1i
			im[i1] = a1i - b1r
			re[i2] = a2r + b2i
			im[i2] = a2i - b2r
			re[i3] = a2r - b2i
			im[i3] = a2i + b2r
			re[i4] = a1r - b1i
			im[i4] = a1i + b1r
		}
	}
}
