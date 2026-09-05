package vorbis

import "math"

const floorFirst = 1.0649863e-07

var logFloorSpan = math.Log(1 / floorFirst)

const (
	floorPartitions      = 7
	shortFloorPartitions = 2
	floorWarp            = 1.5
)

func buildFloor1(n2, partitions int) (*floor1, int) {
	rangebits := blockLog(n2)
	posts := partitions * 8
	xs := make([]int, 0, posts+2)
	xs = append(xs, 0, 1<<rangebits)
	seen := map[int]bool{0: true, 1 << rangebits: true}
	for i := 0; i < posts; i++ {
		frac := float64(i+1) / float64(posts)
		p := int(math.Round(float64(n2-1) * math.Pow(frac, floorWarp)))
		if p < 1 {
			p = 1
		}
		for seen[p] {
			p++
		}
		seen[p] = true
		xs = append(xs, p)
	}
	partClass := make([]int, partitions)
	f := &floor1{
		partitionClass:  partClass,
		classDims:       []int{8},
		classSubclasses: []int{0},
		classMasterbook: []int{0},
		classSubbooks:   [][]int{{bookFloorPosts}},
		multiplier:      1,
		rangeVal:        floor1Ranges[0],
		xs:              xs,
	}
	if err := f.computeNeighbors(); err != nil {
		panic("vorbis: floor geometry has duplicate posts: " + err.Error())
	}
	return f, rangebits
}

func ampToY(amp float64) int {
	if amp <= floorFirst {
		return 0
	}
	y := int(math.Round(255 * math.Log(amp/floorFirst) / logFloorSpan))
	if y < 0 {
		return 0
	}
	if y > 255 {
		return 255
	}
	return y
}

func floor1Fit(f *floor1, spec []float32, dst []int, n2 int) {
	count := len(f.xs)
	for si := 0; si < count; si++ {
		i := f.sortOrder[si]
		lo := 0
		if si > 0 {
			lo = f.xs[f.sortOrder[si-1]]
		}
		hi := n2
		if si < count-1 {
			hi = f.xs[f.sortOrder[si+1]]
		}
		if lo < 0 {
			lo = 0
		}
		if hi > n2 {
			hi = n2
		}
		if hi <= lo {
			hi = lo + 1
		}
		var peak float64
		for b := lo; b < hi && b < n2; b++ {
			if v := math.Abs(float64(spec[b])); v > peak {
				peak = v
			}
		}
		dst[i] = ampToY(peak * floorHeadroom)
	}
}

const floorHeadroom = 1.10

func floor1EncodeVals(f *floor1, targets, vals, final []int) {
	count := len(f.xs)
	vals[0], vals[1] = targets[0], targets[1]
	final[0], final[1] = targets[0], targets[1]
	rng := f.rangeVal
	for i := 2; i < count; i++ {
		low, high := f.lowNeighbor[i], f.highNeighbor[i]
		pred := renderPoint(f.xs[low], final[low], f.xs[high], final[high], f.xs[i])
		vals[i] = floor1EncodeVal(pred, targets[i], rng)
		final[i] = floor1DecodeVal(pred, vals[i], rng)
	}
}

func floor1EncodeVal(pred, target, rng int) int {
	if target == pred {
		return 0
	}
	highroom := rng - pred
	lowroom := pred
	room := 2 * lowroom
	if highroom < lowroom {
		room = 2 * highroom
	}
	var val int
	if target > pred {
		val = 2 * (target - pred)
	} else {
		val = 2*(pred-target) - 1
	}
	if val < room {
		return val
	}
	if highroom > lowroom {
		return target - pred + lowroom
	}
	return pred - target + highroom - 1
}

func floor1DecodeVal(pred, val, rng int) int {
	if val == 0 {
		return pred
	}
	highroom := rng - pred
	lowroom := pred
	room := 2 * lowroom
	if highroom < lowroom {
		room = 2 * highroom
	}
	switch {
	case val >= room:
		if highroom > lowroom {
			return val - lowroom + pred
		}
		return pred - val + highroom - 1
	case val&1 == 1:
		return pred - (val+1)/2
	default:
		return pred + val/2
	}
}

func writeFloorData(w *bitWriter, f *floor1, vals []int, book *encBook) {
	w.writeBit(1)
	ilr := ilog(f.rangeVal - 1)
	w.writeBits(uint(ilr), uint32(vals[0]))
	w.writeBits(uint(ilr), uint32(vals[1]))
	for i := 2; i < len(f.xs); i++ {
		book.emit(w, vals[i])
	}
}
