package resample

import (
	"fmt"
	"math"
	"sync"

	"novelhub/pkg/waxflow/dsp/internal/firwin"
	"novelhub/pkg/waxflow/waxerr"
)

type bank struct {
	l, m  int
	tpp   int
	delay int
	coef  []float32
}

type profileSpec struct {
	attenDB  float64
	passEdge float64
	stopEdge float64
}

var profiles = map[Profile]profileSpec{
	HQ:   {attenDB: 115, passEdge: 0.91, stopEdge: 1.09},
	Fast: {attenDB: 72, passEdge: 0.85, stopEdge: 1.15},
}

type bankKey struct {
	l, m int
	p    Profile
}

type bankEntry struct {
	once sync.Once
	b    *bank
	err  error
}

var (
	bankMu    sync.Mutex
	bankCache = map[bankKey]*bankEntry{}
)

func bankFor(inRate, outRate int, p Profile) (*bank, error) {
	g := gcd(inRate, outRate)
	key := bankKey{outRate / g, inRate / g, p}

	bankMu.Lock()
	e, ok := bankCache[key]
	if !ok {
		e = &bankEntry{}
		bankCache[key] = e
	}
	bankMu.Unlock()

	e.once.Do(func() { e.b, e.err = design(key.l, key.m, p) })
	return e.b, e.err
}

func design(l, m int, p Profile) (*bank, error) {
	spec := profiles[p]
	nuC := math.Min(1, float64(l)/float64(m)) / (2 * float64(l))
	deltaOmega := 2 * math.Pi * (spec.stopEdge - spec.passEdge) * nuC

	nf := math.Ceil((spec.attenDB-7.95)/(2.285*deltaOmega)) + 1
	if !(nf > 0 && nf <= maxBankFloats) {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("resample: ratio %d/%d needs a %.0f-tap filter, over the supported bound", l, m, nf))
	}
	n := max(int(nf), 9)
	if n%2 == 0 {
		n++
	}
	beta := 0.1102 * (spec.attenDB - 8.7)

	if l > maxBankFloats {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("resample: ratio %d/%d needs at least %d phases, over the supported bound", l, m, l))
	}
	tpp := (n + l - 1) / l
	total := tpp * l
	if total > maxBankFloats {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("resample: ratio %d/%d needs a %d-tap filter, over the supported bound", l, m, total))
	}

	h := make([]float64, total)
	d := (n - 1) / 2
	i0beta := firwin.BesselI0(beta)
	var sum float64
	for i := 0; i < n; i++ {
		x := float64(i - d)
		t := x / float64(d)
		w := firwin.BesselI0(beta*math.Sqrt(1-t*t)) / i0beta
		h[i] = firwin.Sinc(2*nuC*x) * w
		sum += h[i]
	}
	scale := float64(l) / sum

	coef := make([]float32, total)
	for p := 0; p < l; p++ {
		for j := 0; j < tpp; j++ {
			coef[p*tpp+j] = float32(h[p+(tpp-1-j)*l] * scale)
		}
	}
	return &bank{l: l, m: m, tpp: tpp, delay: d, coef: coef}, nil
}
