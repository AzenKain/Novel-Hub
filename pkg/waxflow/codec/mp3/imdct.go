package mp3

func imdctSubband(in []float32, bt int, out *[36]float32) {
	in = in[:18]
	if bt == blockShort {
		*out = [36]float32{}
		win := &imdctWinF[blockShort]
		for w := 0; w < 3; w++ {
			var x [12]float32
			for p := 0; p < 3; p++ {
				var sum float32
				for m := 0; m < 6; m++ {
					sum += in[w+3*m] * cosN12f[m][p]
				}
				x[p] = sum
				x[5-p] = -sum
			}
			for p := 6; p < 9; p++ {
				var sum float32
				for m := 0; m < 6; m++ {
					sum += in[w+3*m] * cosN12f[m][p]
				}
				x[p] = sum
				x[17-p] = sum
			}
			for p := 0; p < 12; p++ {
				out[6*w+p+6] += x[p] * win[p]
			}
		}
		return
	}
	win := &imdctWinF[bt]
	var x [36]float32
	for p := 0; p < 9; p++ {
		var sum float32
		for m := 0; m < 18; m++ {
			sum += in[m] * cosN36f[m][p]
		}
		x[p] = sum
		x[17-p] = -sum
	}
	for p := 18; p < 27; p++ {
		var sum float32
		for m := 0; m < 18; m++ {
			sum += in[m] * cosN36f[m][p]
		}
		x[p] = sum
		x[53-p] = sum
	}
	for p := 0; p < 36; p++ {
		out[p] = x[p] * win[p]
	}
}

func (d *Decoder) hybrid(gi *grInfo, g *granule, ch int, nLongBands int) {
	spec := &g.spec[ch]
	store := &d.store[ch]
	var out [36]float32
	for sb := 0; sb < 32; sb++ {
		bt := gi.blockType
		if sb < nLongBands {
			bt = blockNormal
		}
		imdctSubband(spec[sb*18:sb*18+18], bt, &out)
		for i := 0; i < 18; i++ {
			spec[sb*18+i] = out[i] + store[sb][i]
			store[sb][i] = out[i+18]
		}
	}
	for sb := 1; sb < 32; sb += 2 {
		for i := 1; i < 18; i += 2 {
			spec[sb*18+i] = -spec[sb*18+i]
		}
	}
}
