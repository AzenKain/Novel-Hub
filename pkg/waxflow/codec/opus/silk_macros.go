package opus

// SILK fixed-point arithmetic primitives, ported from libopus silk/macros.h, silk/SigProc_FIX.h, silk/Inlines.h, silk/lin2log.c, and silk/log2lin.c.

import "math/bits"

const (
	silkInt32Max = 0x7FFFFFFF
	silkInt32Min = -0x80000000
	silkInt16Max = 0x7FFF
	silkInt16Min = -0x8000
	silkInt8Max  = 0x7F
	silkInt8Min  = -0x80

	randMultiplier = 196314165
	randIncrement  = 907633515
)

func silkSMULWB(a, b int32) int32 { return int32((int64(a) * int64(int16(b))) >> 16) }

func silkSMLAWB(a, b, c int32) int32 { return int32(int64(a) + ((int64(b) * int64(int16(c))) >> 16)) }

func silkSMULWT(a, b int32) int32 { return int32((int64(a) * int64(b>>16)) >> 16) }

func silkSMLAWT(a, b, c int32) int32 { return int32(int64(a) + ((int64(b) * int64(c>>16)) >> 16)) }

func silkSMULBB(a, b int32) int32 { return int32(int16(a)) * int32(int16(b)) }

func silkSMLABB(a, b, c int32) int32 { return a + int32(int16(b))*int32(int16(c)) }

func silkSMULBT(a, b int32) int32 { return int32(int16(a)) * (b >> 16) }

func silkSMLABT(a, b, c int32) int32 { return a + int32(int16(b))*(c>>16) }

func silkSMULWW(a, b int32) int32 { return int32((int64(a) * int64(b)) >> 16) }

func silkSMLAWW(a, b, c int32) int32 { return int32(int64(a) + ((int64(b) * int64(c)) >> 16)) }

func silkSMULTT(a, b int32) int32 { return (a >> 16) * (b >> 16) }

func silkSMLATT(a, b, c int32) int32 { return a + (b>>16)*(c>>16) }

func silkSMULL(a, b int32) int64 { return int64(a) * int64(b) }

func silkSMMUL(a, b int32) int32 { return int32((int64(a) * int64(b)) >> 32) }

func silkMLA(a, b, c int32) int32 { return a + b*c }

func silkADD32ovflw(a, b int32) int32 { return int32(uint32(a) + uint32(b)) }
func silkSUB32ovflw(a, b int32) int32 { return int32(uint32(a) - uint32(b)) }

func silkMLAovflw(a, b, c int32) int32 { return silkADD32ovflw(a, int32(uint32(b)*uint32(c))) }

func silkSMLABBovflw(a, b, c int32) int32 {
	return silkADD32ovflw(a, int32(int16(b))*int32(int16(c)))
}

func silkRAND(seed int32) int32 { return silkMLAovflw(randIncrement, seed, randMultiplier) }

func silkDIV32_16(a, b int32) int32 { return a / b }
func silkDIV32(a, b int32) int32    { return a / b }

func silkSAT8(a int32) int32 {
	if a > silkInt8Max {
		return silkInt8Max
	}
	if a < silkInt8Min {
		return silkInt8Min
	}
	return a
}

func silkSAT16(a int32) int32 {
	if a > silkInt16Max {
		return silkInt16Max
	}
	if a < silkInt16Min {
		return silkInt16Min
	}
	return a
}

func silkSAT32(a int64) int32 {
	if a > silkInt32Max {
		return silkInt32Max
	}
	if a < silkInt32Min {
		return silkInt32Min
	}
	return int32(a)
}

func silkADDSAT32(a, b int32) int32 { return silkSAT32(int64(a) + int64(b)) }
func silkSUBSAT32(a, b int32) int32 { return silkSAT32(int64(a) - int64(b)) }

func silkLSHIFT32(a int32, shift int) int32    { return int32(uint32(a) << uint(shift)) }
func silkLSHIFTovflw(a int32, shift int) int32 { return int32(uint32(a) << uint(shift)) }
func silkRSHIFT(a int32, shift int) int32      { return a >> uint(shift) }
func silkLSHIFT(a int32, shift int) int32      { return silkLSHIFT32(a, shift) }

func silkLIMIT(a, l1, l2 int32) int32 {
	if l1 > l2 {
		if a > l1 {
			return l1
		}
		if a < l2 {
			return l2
		}
		return a
	}
	if a > l2 {
		return l2
	}
	if a < l1 {
		return l1
	}
	return a
}

func silkLSHIFTSAT32(a int32, shift int) int32 {
	return silkLSHIFT32(silkLIMIT(a, silkInt32Min>>uint(shift), silkInt32Max>>uint(shift)), shift)
}

func silkRSHIFTROUND(a int32, shift int) int32 {
	if shift == 1 {
		return (a >> 1) + (a & 1)
	}
	return ((a >> uint(shift-1)) + 1) >> 1
}

func silkRSHIFTROUND64(a int64, shift int) int64 {
	if shift == 1 {
		return (a >> 1) + (a & 1)
	}
	return ((a >> uint(shift-1)) + 1) >> 1
}

func silkMinInt(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func silkMaxInt(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func silkAbs32(a int32) int32 {
	if a > 0 {
		return a
	}
	return -a
}

func silkSign(a int32) int32 {
	if a > 0 {
		return 1
	}
	if a < 0 {
		return -1
	}
	return 0
}

func silkCLZ32(x int32) int32 {
	if x == 0 {
		return 32
	}
	return int32(32 - bits.Len32(uint32(x)))
}

func silkROR32(a int32, rot int) int32 { return int32(bits.RotateLeft32(uint32(a), -rot)) }

func silkCLZFrac(in int32) (lz, fracQ7 int32) {
	lz = silkCLZ32(in)
	fracQ7 = silkROR32(in, int(24-lz)) & 0x7F
	return
}

func silkSQRTAPPROX(x int32) int32 {
	if x <= 0 {
		return 0
	}
	lz, fracQ7 := silkCLZFrac(x)
	var y int32
	if lz&1 != 0 {
		y = 32768
	} else {
		y = 46214
	}
	y >>= uint(lz >> 1)
	y = silkSMLAWB(y, y, silkSMULBB(213, fracQ7))
	return y
}

func silkDIV32varQ(a32, b32 int32, Qres int) int32 {
	aHeadrm := int(silkCLZ32(silkAbs32(a32))) - 1
	a32nrm := silkLSHIFT(a32, aHeadrm)
	bHeadrm := int(silkCLZ32(silkAbs32(b32))) - 1
	b32nrm := silkLSHIFT(b32, bHeadrm)
	b32inv := silkDIV32_16(silkInt32Max>>2, silkRSHIFT(b32nrm, 16))
	result := silkSMULWB(a32nrm, b32inv)
	a32nrm = silkSUB32ovflw(a32nrm, silkLSHIFTovflw(silkSMMUL(b32nrm, result), 3))
	result = silkSMLAWB(result, a32nrm, b32inv)
	lshift := 29 + aHeadrm - bHeadrm - Qres
	if lshift < 0 {
		return silkLSHIFTSAT32(result, -lshift)
	}
	if lshift < 32 {
		return silkRSHIFT(result, lshift)
	}
	return 0
}

func silkINVERSE32varQ(b32 int32, Qres int) int32 {
	bHeadrm := int(silkCLZ32(silkAbs32(b32))) - 1
	b32nrm := silkLSHIFT(b32, bHeadrm)
	b32inv := silkDIV32_16(silkInt32Max>>2, silkRSHIFT(b32nrm, 16))
	result := silkLSHIFT(b32inv, 16)
	errQ32 := silkLSHIFT(int32(1)<<29-silkSMULWB(b32nrm, b32inv), 3)
	result = silkSMLAWW(result, errQ32, b32inv)
	lshift := 61 - bHeadrm - Qres
	if lshift <= 0 {
		return silkLSHIFTSAT32(result, -lshift)
	}
	if lshift < 32 {
		return silkRSHIFT(result, lshift)
	}
	return 0
}

func silkLin2Log(inLin int32) int32 {
	lz, fracQ7 := silkCLZFrac(inLin)
	return silkSMLAWB(fracQ7, silkMLA(0, fracQ7, 128-fracQ7), 179) + silkLSHIFT(31-lz, 7)
}

func silkLog2Lin(inLogQ7 int32) int32 {
	if inLogQ7 < 0 {
		return 0
	}
	if inLogQ7 >= 3967 {
		return silkInt32Max
	}
	out := silkLSHIFT(1, int(silkRSHIFT(inLogQ7, 7)))
	fracQ7 := inLogQ7 & 0x7F
	if inLogQ7 < 2048 {
		out = out + silkRSHIFT(silkMLA(0, out, silkSMLAWB(fracQ7, silkSMULBB(fracQ7, 128-fracQ7), -174)), 7)
	} else {
		out = silkMLA(out, silkRSHIFT(out, 7), silkSMLAWB(fracQ7, silkSMULBB(fracQ7, 128-fracQ7), -174))
	}
	return out
}
