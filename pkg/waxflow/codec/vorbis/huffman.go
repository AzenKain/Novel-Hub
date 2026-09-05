package vorbis

func huffmanLengths(freq []float64) []uint8 {
	n := len(freq)
	lengths := make([]uint8, n)
	if n == 0 {
		return lengths
	}
	if n == 1 {
		lengths[0] = 1
		return lengths
	}

	type node struct {
		weight      float64
		left, right int
		active      bool
	}
	nodes := make([]node, 0, 2*n)
	for i := 0; i < n; i++ {
		w := freq[i]
		if w <= 0 {
			w = 1e-12
		}
		nodes = append(nodes, node{weight: w, left: -1, right: -1, active: true})
	}

	smallest := func() int {
		best := -1
		for i := range nodes {
			if nodes[i].active && (best < 0 || nodes[i].weight < nodes[best].weight) {
				best = i
			}
		}
		return best
	}

	active := n
	for active > 1 {
		a := smallest()
		nodes[a].active = false
		b := smallest()
		nodes[b].active = false
		nodes = append(nodes, node{weight: nodes[a].weight + nodes[b].weight, left: a, right: b, active: true})
		active--
	}

	root := len(nodes) - 1
	var walk func(idx, depth int)
	walk = func(idx, depth int) {
		if nodes[idx].left < 0 {
			if idx < n {
				d := depth
				if d < 1 {
					d = 1
				}
				if d > maxCodewordLen {
					d = maxCodewordLen
				}
				lengths[idx] = uint8(d)
			}
			return
		}
		walk(nodes[idx].left, depth+1)
		walk(nodes[idx].right, depth+1)
	}
	walk(root, 0)

	fixKraft(lengths)
	return lengths
}

func fixKraft(lengths []uint8) {
	kraft := func() float64 {
		s := 0.0
		for _, l := range lengths {
			if l > 0 {
				s += 1.0 / float64(int64(1)<<l)
			}
		}
		return s
	}
	for kraft() > 1.0 {
		shortest := -1
		for i, l := range lengths {
			if l == 0 {
				continue
			}
			if shortest < 0 || l < lengths[shortest] {
				shortest = i
			}
		}
		if shortest < 0 || lengths[shortest] >= maxCodewordLen {
			return
		}
		lengths[shortest]++
	}
}
