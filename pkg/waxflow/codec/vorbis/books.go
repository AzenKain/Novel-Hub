package vorbis

//go:generate go test -tags booksgen -run ^TestGenerateBooks$ -count=1

import "math/bits"

const (
	bookFloorPosts = 0
	bookResClass   = 1
	bookResNoise   = 2
	bookResCoarse  = 3
	bookResR1      = 4
	bookResR2      = 5
	bookResR3      = 6
)

const (
	resNoiseDim     = 2
	resNoiseL       = 32
	resNoiseMin     = -2.0
	resNoiseDelta   = 1.0 / 8.0
	resNoiseEntries = resNoiseL * resNoiseL

	resCoarseDim     = 2
	resCoarseL       = 64
	resCoarseMin     = -2.0
	resCoarseDelta   = 1.0 / 16.0
	resCoarseEntries = resCoarseL * resCoarseL

	resRefineDim = 2
	resRefineL   = 5
	resR1Delta   = 1.0 / 64.0
	resR1Min     = -2.0 / 64.0
	resR1Entries = resRefineL * resRefineL
	resR2Delta   = 1.0 / 256.0
	resR2Min     = -2.0 / 256.0
	resR2Entries = resRefineL * resRefineL
	resR3Delta   = 1.0 / 1024.0
	resR3Min     = -2.0 / 1024.0
	resR3Entries = resRefineL * resRefineL
)

func bookSpecs() []bookSpec {
	return []bookSpec{
		bookFloorPosts: floorPostsSpec(),
		bookResClass:   classbookSpec(),
		bookResNoise:   productBookSpec(resNoiseDim, resNoiseL, resNoiseMin, resNoiseDelta, resNoiseLengths),
		bookResCoarse:  productBookSpec(resCoarseDim, resCoarseL, resCoarseMin, resCoarseDelta, resCoarseLengths),
		bookResR1:      productBookSpec(resRefineDim, resRefineL, resR1Min, resR1Delta, resR1Lengths),
		bookResR2:      productBookSpec(resRefineDim, resRefineL, resR2Min, resR2Delta, resR2Lengths),
		bookResR3:      productBookSpec(resRefineDim, resRefineL, resR3Min, resR3Delta, resR3Lengths),
	}
}

var (
	encSpecs = bookSpecs()
	encBooks = buildEncBooks(encSpecs)
)

func buildEncBooks(specs []bookSpec) []*encBook {
	books := make([]*encBook, len(specs))
	for i := range specs {
		books[i] = buildEncBook(specs[i])
	}
	return books
}

func productBookSpec(dim, L int, minimum, delta float64, lengths []uint8) bookSpec {
	mult := make([]uint32, L)
	for i := range mult {
		mult[i] = uint32(i)
	}
	return bookSpec{
		dimensions:   dim,
		lengths:      lengths,
		lookupType:   1,
		minimum:      minimum,
		delta:        delta,
		valueBits:    bits.Len(uint(L - 1)),
		multiplicand: mult,
	}
}

func floorPostsSpec() bookSpec {
	lengths := make([]uint8, 256)
	for i := range lengths {
		lengths[i] = 8
	}
	return bookSpec{dimensions: 1, lengths: lengths}
}

func classbookSpec() bookSpec {
	return bookSpec{dimensions: 1, lengths: huffmanLengths([]float64{0.34, 0.10, 0.22, 0.16, 0.12, 0.06})}
}
