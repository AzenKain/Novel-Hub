package vorbis

type residue struct {
	kind      int
	begin     int
	end       int
	partSize  int
	classes   int
	classbook int
	books     [][]int
	maxPass   int
}

func parseResidue(r *bitReader, numBooks int) (residue, error) {
	var res residue
	typ := int(r.read(16))
	if typ < 0 || typ > 2 {
		return res, malformed("residue type %d", typ)
	}
	res.kind = typ
	res.begin = int(r.read(24))
	res.end = int(r.read(24))
	res.partSize = int(r.read(24)) + 1
	res.classes = int(r.read(6)) + 1
	res.classbook = int(r.read(8))
	if res.classbook >= numBooks {
		return res, malformed("residue classbook %d of %d", res.classbook, numBooks)
	}
	if res.begin > res.end {
		return res, malformed("residue begin %d > end %d", res.begin, res.end)
	}
	cascade := make([]int, res.classes)
	for i := range cascade {
		highBits := 0
		lowBits := int(r.read(3))
		if r.bit() == 1 {
			highBits = int(r.read(5))
		}
		cascade[i] = highBits*8 + lowBits
	}
	res.books = make([][]int, res.classes)
	for i := 0; i < res.classes; i++ {
		res.books[i] = make([]int, 8)
		for j := 0; j < 8; j++ {
			res.books[i][j] = -1
			if cascade[i]&(1<<uint(j)) != 0 {
				b := int(r.read(8))
				if b >= numBooks {
					return res, malformed("residue book %d of %d", b, numBooks)
				}
				res.books[i][j] = b
				if j+1 > res.maxPass {
					res.maxPass = j + 1
				}
			}
		}
	}
	if r.eof {
		return res, malformed("residue header truncated")
	}
	return res, nil
}

func (res *residue) decode(r *bitReader, cbs []codebook, out [][]float32, doNotDecode []bool, size int, vec []float32) error {
	ch := len(out)
	classbook := &cbs[res.classbook]
	classwords := classbook.dimensions
	if classwords <= 0 || res.classes <= 0 {
		return malformed("residue classbook has no dimensions")
	}

	if res.kind == 2 {
		return res.decodeType2(r, cbs, out, doNotDecode, size, classbook, classwords, vec)
	}

	begin, end := res.begin, res.end
	if end > size {
		end = size
	}
	if begin > end {
		begin = end
	}
	nToRead := end - begin
	partRead := nToRead / res.partSize
	if partRead == 0 {
		return nil
	}
	class := make([][]int, ch)
	for j := range class {
		if !doNotDecode[j] {
			class[j] = make([]int, partRead)
		}
	}
	for pass := 0; pass < res.maxPass; pass++ {
		pcount := 0
		for pcount < partRead {
			if pass == 0 {
				for j := 0; j < ch; j++ {
					if doNotDecode[j] {
						continue
					}
					q, err := classbook.decodeScalar(r)
					if err != nil {
						return residueEOF(err)
					}
					for i := classwords - 1; i >= 0; i-- {
						if pcount+i < partRead {
							class[j][pcount+i] = q % res.classes
						}
						q /= res.classes
					}
				}
			}
			for i := 0; i < classwords && pcount < partRead; i, pcount = i+1, pcount+1 {
				for j := 0; j < ch; j++ {
					if doNotDecode[j] {
						continue
					}
					book := res.books[class[j][pcount]][pass]
					if book < 0 {
						continue
					}
					off := begin + pcount*res.partSize
					if err := res.partition(r, &cbs[book], out[j], off, vec); err != nil {
						return residueEOF(err)
					}
				}
			}
		}
	}
	return nil
}

func (res *residue) decodeType2(r *bitReader, cbs []codebook, out [][]float32, doNotDecode []bool, size int, classbook *codebook, classwords int, vec []float32) error {
	ch := len(out)
	anyDecode := false
	for _, skip := range doNotDecode {
		if !skip {
			anyDecode = true
			break
		}
	}
	if !anyDecode {
		return nil
	}
	begin, end := res.begin, res.end
	if end > ch*size {
		end = ch * size
	}
	if begin > end {
		begin = end
	}
	nToRead := end - begin
	partRead := nToRead / res.partSize
	if partRead == 0 {
		return nil
	}
	class := make([]int, partRead)
	for pass := 0; pass < res.maxPass; pass++ {
		pcount := 0
		for pcount < partRead {
			if pass == 0 {
				q, err := classbook.decodeScalar(r)
				if err != nil {
					return residueEOF(err)
				}
				for i := classwords - 1; i >= 0; i-- {
					if pcount+i < partRead {
						class[pcount+i] = q % res.classes
					}
					q /= res.classes
				}
			}
			for i := 0; i < classwords && pcount < partRead; i, pcount = i+1, pcount+1 {
				book := res.books[class[pcount]][pass]
				if book >= 0 {
					off := begin + pcount*res.partSize
					if err := res.partitionInterleaved(r, &cbs[book], out, off, ch, vec); err != nil {
						return residueEOF(err)
					}
				}
			}
		}
	}
	return nil
}

func (res *residue) partition(r *bitReader, book *codebook, target []float32, off int, vec []float32) error {
	dim := book.dimensions
	if res.kind == 0 {
		step := res.partSize / dim
		for i := 0; i < step; i++ {
			entry, err := book.decodeScalar(r)
			if err != nil {
				return err
			}
			book.valueVector(entry, vec[:dim])
			for k := 0; k < dim; k++ {
				pos := off + i + k*step
				if pos >= 0 && pos < len(target) {
					target[pos] += vec[k]
				}
			}
		}
		return nil
	}
	for i := 0; i < res.partSize; {
		entry, err := book.decodeScalar(r)
		if err != nil {
			return err
		}
		book.valueVector(entry, vec[:dim])
		for k := 0; k < dim && i < res.partSize; k, i = k+1, i+1 {
			if pos := off + i; pos >= 0 && pos < len(target) {
				target[pos] += vec[k]
			}
		}
	}
	return nil
}

func (res *residue) partitionInterleaved(r *bitReader, book *codebook, out [][]float32, off, ch int, vec []float32) error {
	dim := book.dimensions
	for i := 0; i < res.partSize; {
		entry, err := book.decodeScalar(r)
		if err != nil {
			return err
		}
		book.valueVector(entry, vec[:dim])
		for k := 0; k < dim && i < res.partSize; k, i = k+1, i+1 {
			p := off + i
			c := p % ch
			bin := p / ch
			if bin >= 0 && bin < len(out[c]) {
				out[c][bin] += vec[k]
			}
		}
	}
	return nil
}

func residueEOF(err error) error {
	if err == errEndOfPacket {
		return nil
	}
	return err
}

func maxDim(cbs []codebook) int {
	m := 1
	for i := range cbs {
		if cbs[i].dimensions > m {
			m = cbs[i].dimensions
		}
	}
	return m
}
