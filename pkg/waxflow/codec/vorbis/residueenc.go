package vorbis

const maxBookVecDim = 8

const maxResPass = 8

func encodeResidueType1(w *bitWriter, r *residue, books []*encBook, residues [][]float32, classes [][]int, n2, magChannel int) {
	ch := len(residues)
	classbook := books[r.classbook]
	var ang []float32
	if magChannel >= 0 && ch == 2 {
		ang = residues[1-magChannel]
	}
	begin, end := r.begin, r.end
	if end > n2 {
		end = n2
	}
	partRead := (end - begin) / r.partSize
	for p := 0; p < partRead; p++ {
		for j := 0; j < ch; j++ {
			classbook.emit(w, classes[j][p])
		}
		for j := 0; j < ch; j++ {
			chain := r.books[classes[j][p]]
			if chain[0] >= 0 {
				emitResidueVectors(w, books[chain[0]], nil, residues[j], begin+p*r.partSize, r.partSize, n2, magAngle(j == magChannel, ang), isLastBook(chain, 0))
			}
		}
	}
	var prevs [maxResPass]*encBook
	for pass := 1; pass < r.maxPass; pass++ {
		for p := 0; p < partRead; p++ {
			for j := 0; j < ch; j++ {
				chain := r.books[classes[j][p]]
				book := chain[pass]
				if book < 0 {
					continue
				}
				np := 0
				for k := 0; k < pass; k++ {
					if chain[k] >= 0 {
						prevs[np] = books[chain[k]]
						np++
					}
				}
				emitResidueVectors(w, books[book], prevs[:np], residues[j], begin+p*r.partSize, r.partSize, n2, magAngle(j == magChannel, ang), isLastBook(chain, pass))
			}
		}
	}
}

func isLastBook(chain []int, pass int) bool {
	for k := pass + 1; k < len(chain); k++ {
		if chain[k] >= 0 {
			return false
		}
	}
	return true
}

func magAngle(isMag bool, ang []float32) []float32 {
	if isMag {
		return ang
	}
	return nil
}

func emitResidueVectors(w *bitWriter, book *encBook, prevs []*encBook, resid []float32, off, partSize, n2 int, ang []float32, lastPass bool) {
	dim := book.dimensions
	var vec [maxBookVecDim]float64
	var sp [maxBookVecDim]bool
	for i := 0; i < partSize; i += dim {
		for k := 0; k < dim; k++ {
			bin := off + i + k
			v := 0.0
			inRange := bin >= 0 && bin < n2
			if inRange {
				v = float64(resid[bin])
			}
			orig := v
			for _, pb := range prevs {
				v -= pb.latValue(pb.latIndex(v))
			}
			sp[k] = lastPass && v == orig && ang != nil && inRange &&
				book.latValue(book.latIndex(v)) == 0 &&
				cascadeValue(book, prevs, float64(ang[bin])) != 0
			vec[k] = v
		}
		book.emit(w, book.vectorEntry(vec[:dim], sp[:dim]))
	}
}

func cascadeValue(book *encBook, prevs []*encBook, x float64) float64 {
	total := 0.0
	for _, pb := range prevs {
		r := pb.latValue(pb.latIndex(x))
		total += r
		x -= r
	}
	return total + book.latValue(book.latIndex(x))
}

func normalizeResidue(spec, curve, dst []float32, n2 int) {
	for i := 0; i < n2; i++ {
		dst[i] = spec[i] / curve[i]
	}
}
