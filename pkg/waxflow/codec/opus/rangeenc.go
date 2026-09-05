package opus

// rangeEncoder is Opus's entropy encoder (RFC 6716 section 4.1), the inverse of rangeDecoder: range-coded symbols accumulate from the front of the packet and raw bits from the back.
type rangeEncoder struct {
	buf     []byte
	storage int
	offs    int
	endOffs int
	endWin  uint32
	nEnd    int
	nbits   int
	rng     uint32
	val     uint32
	ext     uint32
	rem     int
	err     bool
}

const ecCodeShift = ecCodeBits - ecSymBits - 1

func newRangeEncoder(buf []byte) *rangeEncoder {
	return &rangeEncoder{
		buf:     buf,
		storage: len(buf),
		nbits:   ecCodeBits + 1,
		rng:     ecCodeTop,
		rem:     -1,
	}
}

func (e *rangeEncoder) snapshot() rangeEncoder { return *e }

func (e *rangeEncoder) restore(s *rangeEncoder) { *e = *s }

func (e *rangeEncoder) tailBytes(from *rangeEncoder, dst []byte) []byte {
	return append(dst[:0], e.buf[from.offs:from.storage]...)
}

func (e *rangeEncoder) restoreTail(from *rangeEncoder, saved []byte) {
	copy(e.buf[from.offs:from.storage], saved)
}

func (e *rangeEncoder) writeByte(v byte) {
	if e.offs+e.endOffs >= e.storage {
		e.err = true
		return
	}
	e.buf[e.offs] = v
	e.offs++
}

func (e *rangeEncoder) writeByteAtEnd(v byte) {
	if e.offs+e.endOffs >= e.storage {
		e.err = true
		return
	}
	e.endOffs++
	e.buf[e.storage-e.endOffs] = v
}

func (e *rangeEncoder) carryOut(c int) {
	if c != ecSymMax {
		carry := c >> ecSymBits
		if e.rem >= 0 {
			e.writeByte(byte(e.rem + carry))
		}
		if e.ext > 0 {
			sym := byte((ecSymMax + carry) & ecSymMax)
			for {
				e.writeByte(sym)
				e.ext--
				if e.ext == 0 {
					break
				}
			}
		}
		e.rem = c & ecSymMax
	} else {
		e.ext++
	}
}

func (e *rangeEncoder) normalize() {
	for e.rng <= ecCodeBot {
		e.carryOut(int(e.val >> ecCodeShift))
		e.val = (e.val << ecSymBits) & (ecCodeTop - 1)
		e.rng <<= ecSymBits
		e.nbits += ecSymBits
	}
}

// encode codes a symbol with cumulative range [fl, fh) of total ft (RFC 6716 4.1.2), the inverse of rangeDecoder.decode+update.
func (e *rangeEncoder) encode(fl, fh, ft uint32) {
	r := e.rng / ft
	if fl > 0 {
		e.val += e.rng - r*(ft-fl)
		e.rng = r * (fh - fl)
	} else {
		e.rng -= r * (ft - fh)
	}
	e.normalize()
}

// encodeBin is encode with ft == 1<<bits (RFC 6716 4.1.3.1).
func (e *rangeEncoder) encodeBin(fl, fh uint32, bits uint) {
	r := e.rng >> bits
	if fl > 0 {
		e.val += e.rng - r*((1<<bits)-fl)
		e.rng = r * (fh - fl)
	} else {
		e.rng -= r * ((1 << bits) - fh)
	}
	e.normalize()
}

// encodeBitLogp codes one bit whose probability of being one is 2^-logp (RFC 6716 4.1.3.2).
func (e *rangeEncoder) encodeBitLogp(val int, logp uint) {
	r := e.rng
	l := e.val
	s := r >> logp
	r -= s
	if val != 0 {
		e.val = l + r
		e.rng = s
	} else {
		e.rng = r
	}
	e.normalize()
}

// encodeICDF codes symbol s from an inverse cumulative distribution scaled to 2^ftb (RFC 6716 4.1.3.3).
func (e *rangeEncoder) encodeICDF(s int, icdf []byte, ftb uint) {
	r := e.rng >> ftb
	if s > 0 {
		e.val += e.rng - r*uint32(icdf[s-1])
		e.rng = r * uint32(icdf[s-1]-icdf[s])
	} else {
		e.rng -= r * uint32(icdf[s])
	}
	e.normalize()
}

// encodeUint codes a uniformly distributed integer fl in [0, ft) (RFC 6716 4.1.5).
func (e *rangeEncoder) encodeUint(fl, ft uint32) {
	ft--
	ftb := ilog(ft)
	if ftb > ecUintBits {
		ftb -= ecUintBits
		t := (ft >> uint(ftb)) + 1
		e.encode(fl>>uint(ftb), (fl>>uint(ftb))+1, t)
		e.encodeRawBits(fl&((1<<uint(ftb))-1), uint(ftb))
	} else {
		e.encode(fl, fl+1, ft+1)
	}
}

// encodeRawBits writes bits raw toward the back of the packet (RFC 6716 4.1.4), the inverse of rangeDecoder.decodeRawBits.
func (e *rangeEncoder) encodeRawBits(val uint32, bits uint) {
	window := e.endWin
	used := e.nEnd
	if used+int(bits) > ecWindow {
		for {
			e.writeByteAtEnd(byte(window & ecSymMax))
			window >>= ecSymBits
			used -= ecSymBits
			if used < ecSymBits {
				break
			}
		}
	}
	window |= val << uint(used)
	used += int(bits)
	e.endWin = window
	e.nEnd = used
	e.nbits += int(bits)
}

func (e *rangeEncoder) patchInitialBits(val uint32, nbits uint) {
	shift := ecSymBits - int(nbits)
	mask := uint32((1<<nbits)-1) << uint(shift)
	switch {
	case e.offs > 0:
		e.buf[0] = byte((uint32(e.buf[0]) &^ mask) | val<<uint(shift))
	case e.rem >= 0:
		e.rem = int((uint32(e.rem) &^ mask) | val<<uint(shift))
	case e.rng <= ecCodeTop>>nbits:
		e.val = (e.val &^ (mask << ecCodeShift)) | val<<uint(ecCodeShift+shift)
	default:
		e.err = true
	}
}

// tell returns the number of whole bits emitted so far (RFC 6716 4.1.6).
func (e *rangeEncoder) tell() int { return e.nbits - ilog(e.rng) }

// tellFrac returns bits emitted in eighth-bit units (RFC 6716 4.1.6).
func (e *rangeEncoder) tellFrac() int {
	return ecTellFrac(e.nbits, e.rng)
}

func (e *rangeEncoder) shrink(size int) {
	copy(e.buf[size-e.endOffs:size], e.buf[e.storage-e.endOffs:e.storage])
	e.storage = size
}

func (e *rangeEncoder) done() {
	l := ecCodeBits - ilog(e.rng)
	msk := uint32(ecCodeTop-1) >> uint(l)
	end := (e.val + msk) &^ msk
	if end|msk >= e.val+e.rng {
		l++
		msk >>= 1
		end = (e.val + msk) &^ msk
	}
	for l > 0 {
		e.carryOut(int(end >> ecCodeShift))
		end = (end << ecSymBits) & (ecCodeTop - 1)
		l -= ecSymBits
	}
	if e.rem >= 0 || e.ext > 0 {
		e.carryOut(0)
	}
	window := e.endWin
	used := e.nEnd
	for used >= ecSymBits {
		e.writeByteAtEnd(byte(window & ecSymMax))
		window >>= ecSymBits
		used -= ecSymBits
	}
	if !e.err {
		for i := e.offs; i < e.storage-e.endOffs; i++ {
			e.buf[i] = 0
		}
		if used > 0 {
			if e.endOffs >= e.storage {
				e.err = true
			} else {
				l = -l
				if e.offs+e.endOffs >= e.storage && l < used {
					window &= (1 << uint(l)) - 1
					e.err = true
				}
				e.buf[e.storage-e.endOffs-1] |= byte(window)
			}
		}
	}
}

func (e *rangeEncoder) payload() []byte { return e.buf[:e.storage] }
