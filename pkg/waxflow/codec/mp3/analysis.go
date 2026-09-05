package mp3

import "math"

var (
	enwindow [512]float64
	anaCos   [32][64]float64
)

const mdctScale = 1.0 / 9.0

func init() {
	for i := range enwindow {
		enwindow[i] = float64(synthD[i]) / 32
	}
	for sb := 0; sb < 32; sb++ {
		for k := 0; k < 64; k++ {
			anaCos[sb][k] = math.Cos(float64((2*sb+1)*(k-16)) * math.Pi / 64)
		}
	}
}

type analyzer struct {
	fifo    [512]float64
	overlap [32][18]float64
}

func (a *analyzer) reset() {
	a.fifo = [512]float64{}
	a.overlap = [32][18]float64{}
}

func (a *analyzer) subband(in []float32, out *[32]float64) {
	f := &a.fifo
	copy(f[32:], f[:512-32])
	for i := 0; i < 32; i++ {
		f[i] = float64(in[31-i])
	}
	var z [64]float64
	for i := 0; i < 64; i++ {
		s := 0.0
		for j := 0; j < 8; j++ {
			s += f[i+64*j] * enwindow[i+64*j]
		}
		z[i] = s
	}
	for sb := 0; sb < 32; sb++ {
		m := &anaCos[sb]
		s := 0.0
		for k := 0; k < 64; k++ {
			s += z[k] * m[k]
		}
		out[sb] = s
	}
}

func (a *analyzer) granuleMDCT(pcm []float32, xr *[576]float32) {
	var sb [32][18]float64
	var s [32]float64
	for ss := 0; ss < 18; ss++ {
		a.subband(pcm[32*ss:32*ss+32], &s)
		for j := 0; j < 32; j++ {
			sb[j][ss] = s[j]
		}
	}

	for j := 1; j < 32; j += 2 {
		for p := 1; p < 18; p += 2 {
			sb[j][p] = -sb[j][p]
		}
	}

	win := &imdctWinF[blockNormal]
	for j := 0; j < 32; j++ {
		var z [36]float64
		for p := 0; p < 18; p++ {
			z[p] = a.overlap[j][p] * float64(win[p])
			z[p+18] = sb[j][p] * float64(win[p+18])
		}
		a.overlap[j] = sb[j]

		for m := 0; m < 18; m++ {
			acc := 0.0
			for p := 0; p < 36; p++ {
				acc += z[p] * float64(cosN36f[m][p])
			}
			xr[j*18+m] = float32(acc * mdctScale)
		}
	}

	forwardAlias(xr)
}

func forwardAlias(xr *[576]float32) {
	for sb := 1; sb <= 31; sb++ {
		edge := 18 * sb
		for i := 0; i < 8; i++ {
			lo, hi := edge-1-i, edge+i
			a, b := float64(xr[lo]), float64(xr[hi])
			xr[lo] = float32(a*csTab[i] + b*caTab[i])
			xr[hi] = float32(b*csTab[i] - a*caTab[i])
		}
	}
}
