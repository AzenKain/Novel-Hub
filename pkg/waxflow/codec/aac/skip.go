package aac

func skipDSE(r *bitReader) {
	r.read(4)
	align := r.bit()
	count := int(r.read(8))
	if count == 255 {
		count += int(r.read(8))
	}
	if align != 0 {
		r.byteAlign()
	}
	r.skip(count * 8)
}

func skipFIL(r *bitReader) {
	count := int(r.read(4))
	if count == 15 {
		count += int(r.read(8)) - 1
	}
	r.skip(count * 8)
}

func skipPCE(r *bitReader) {
	r.read(4)
	r.read(2)
	r.read(4)
	numFront := int(r.read(4))
	numSide := int(r.read(4))
	numBack := int(r.read(4))
	numLFE := int(r.read(2))
	numAssoc := int(r.read(3))
	numCC := int(r.read(4))
	if r.bit() != 0 {
		r.read(4)
	}
	if r.bit() != 0 {
		r.read(4)
	}
	if r.bit() != 0 {
		r.read(3)
	}
	for i := 0; i < numFront+numSide+numBack; i++ {
		r.read(5)
	}
	for i := 0; i < numLFE+numAssoc; i++ {
		r.read(4)
	}
	for i := 0; i < numCC; i++ {
		r.read(5)
	}
	r.byteAlign()
	comment := int(r.read(8))
	r.skip(comment * 8)
}
