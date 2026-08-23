package doc

import (
	"encoding/binary"
	"math"
	"strings"
)

type charFormat struct {
	bold         bool
	italic       bool
	underline    uint8
	strike       bool
	doubleStrike bool
	allCaps      bool
	smallCaps    bool
	superScript  bool
	subScript    bool
	imprint      bool
	emboss       bool
	hidden       bool
	fontSize     uint16
	fontColor    uint32
	highlight    uint8
	fontIndex    uint16
}

type paraFormat struct {
	alignment       uint8
	leftIndent      int16
	rightIndent     int16
	firstLine       int16
	spaceBefore     uint16
	spaceAfter      uint16
	lineSpacing     int16
	lineRule        uint8
	keepTogether    bool
	keepNext        bool
	outlineLevel    uint8
	pageBreakBefore bool
	inTable         bool
}

const (
	sprmCBold          uint16 = 0x0835
	sprmCItalic        uint16 = 0x0836
	sprmCUnderline     uint16 = 0x0837
	sprmCStrike        uint16 = 0x0838
	sprmCAllCaps       uint16 = 0x0839
	sprmCSmallCaps     uint16 = 0x083A
	sprmCHidden        uint16 = 0x083B
	sprmCSuperscript   uint16 = 0x083C
	sprmCSubscript     uint16 = 0x083D
	sprmCImprint       uint16 = 0x084F
	sprmCEmboss        uint16 = 0x084E
	sprmCFontSize      uint16 = 0x0840
	sprmCKern          uint16 = 0x0842
	sprmCFontIndex     uint16 = 0x0841
	sprmCColor         uint16 = 0x0845
	sprmCHighlight     uint16 = 0x0846
	sprmCCharScaling   uint16 = 0x0850
	sprmCNoProof       uint16 = 0x0857
	sprmCDoubleStrike  uint16 = 0x0858
	sprmCRevMarkDel    uint16 = 0x0859
	sprmCRevMarkIns    uint16 = 0x085A

	sprmPJc              uint16 = 0x2403
	sprmPLeftIndent      uint16 = 0x2407
	sprmPRightIndent     uint16 = 0x240D
	sprmPFirstLine       uint16 = 0x2408
	sprmPSpaceBefore     uint16 = 0x240B
	sprmPSpaceAfter      uint16 = 0x240C
	sprmPLineSpacing     uint16 = 0x2411
	sprmPKeepTogether    uint16 = 0x2405
	sprmPKeepNext        uint16 = 0x2406
	sprmPPageBreakBefore uint16 = 0x2417
	sprmPOutlineLevel    uint16 = 0x2423
	sprmPFInTable        uint16 = 0x2416
)

type textPiece struct {
	text    string
	cpStart int
}

type docStreams struct {
	word  []byte
	table []byte
}

func openStreams(word []byte, streams map[string][]byte) docStreams {
	tblName := "0Table"
	if len(word) > 0x0c && binary.LittleEndian.Uint16(word[0x0a:0x0c])&0x0200 != 0 {
		tblName = "1Table"
	}
	tbl := streams[tblName]
	if len(tbl) == 0 {
		if tblName == "0Table" {
			tbl = streams["1Table"]
		} else {
			tbl = streams["0Table"]
		}
	}
	return docStreams{word: word, table: tbl}
}

func fibOffsets(word []byte) (fcChpx, cbChpx, fcPapx, cbPapx uint32) {
	if len(word) < 0x10a {
		return 0, 0, 0, 0
	}
	fcChpx = binary.LittleEndian.Uint32(word[0xFA:0xFE])
	cbChpx = binary.LittleEndian.Uint32(word[0xFE:0x102])
	fcPapx = binary.LittleEndian.Uint32(word[0x102:0x106])
	cbPapx = binary.LittleEndian.Uint32(word[0x106:0x10A])
	return
}

func (ds docStreams) readFKPPage(pn int) []byte {
	if pn < 0 {
		return nil
	}
	off := pn * 512
	if off+512 <= len(ds.table) {
		return ds.table[off : off+512]
	}
	if off+512 <= len(ds.word) {
		return ds.word[off : off+512]
	}
	return nil
}

func detectCPScale(word []byte, charMap map[int]charFormat, paraMap map[int]paraFormat) (base int, scale int) {
	if len(word) < 0x20 {
		return 0, 1
	}
	fcMin := int(binary.LittleEndian.Uint32(word[0x18:0x1c]))
	maxKey := 0
	for k := range charMap {
		if k > maxKey {
			maxKey = k
		}
	}
	for k := range paraMap {
		if k > maxKey {
			maxKey = k
		}
	}
	if maxKey == 0 {
		return 0, 1
	}
	totalCPs := int(binary.LittleEndian.Uint32(word[0x4c:0x50]))
	if totalCPs == 0 {
		totalCPs = maxKey - fcMin + 1
	}
	if fcMin > 0 && maxKey >= fcMin && maxKey-fcMin > totalCPs*3/2 {
		return fcMin, 2
	}
	return 0, 1
}

func keyCP(base, scale, pieceCP int) int {
	if scale == 2 && base > 0 {
		return base + (pieceCP*2)
	}
	return pieceCP
}

func charMapLookup(charMap map[int]charFormat, cpBase, cpScale, cp int) (charFormat, bool) {
	f, ok := charMap[keyCP(cpBase, cpScale, cp)]
	return f, ok
}

func paraMapLookup(paraMap map[int]paraFormat, cpBase, cpScale, cp int) (paraFormat, bool) {
	f, ok := paraMap[keyCP(cpBase, cpScale, cp)]
	return f, ok
}

func buildCharFormatMap(word []byte, ds docStreams) map[int]charFormat {
	fc, cb, _, _ := fibOffsets(word)
	if cb == 0 || int(fc+cb) > len(ds.table) {
		return nil
	}
	plc := ds.table[fc : fc+cb]
	n := plcEntryCount(plc)
	if n <= 0 {
		return nil
	}
	cpData := plc[:n*4]
	fkpNums := plc[n*4 : n*4+n*4]
	charMap := make(map[int]charFormat)
	for i := 0; i < n; i++ {
		cpMin := int(binary.LittleEndian.Uint32(cpData[i*4 : i*4+4]))
		cpMax := int(binary.LittleEndian.Uint32(cpData[(i+1)*4 : (i+1)*4+4]))
		pn := int(binary.LittleEndian.Uint32(fkpNums[i*4 : i*4+4]))
		page := ds.readFKPPage(pn)
		if page == nil {
			continue
		}
		cpCount := int(page[511])
		if cpCount == 0 || cpCount > 64 {
			continue
		}
		for j := 0; j < cpCount; j++ {
			cpAtJ := int(binary.LittleEndian.Uint32(page[j*4 : j*4+4]))
			offsetWord := int(page[(cpCount+1)*4+j])
			if offsetWord == 0 {
				continue
			}
			byteOff := offsetWord * 2
			if byteOff >= 510 {
				continue
			}
			sprmSize := int(page[byteOff])
			if sprmSize == 0 || byteOff+1+sprmSize > 510 {
				continue
			}
			sprms := page[byteOff+1 : byteOff+1+sprmSize]
			f := decodeCharSprms(sprms)
			nextCp := cpMax
			if j+1 <= cpCount && (j+1)*4+4 <= 512 {
				next := int(binary.LittleEndian.Uint32(page[(j+1)*4 : (j+1)*4+4]))
				if next > 0 {
					nextCp = next
				}
			}
			for cp := cpAtJ; cp < nextCp && cp >= cpMin && cp < cpMax; cp++ {
				charMap[cp] = f
			}
		}
	}
	return charMap
}

func buildParaFormatMap(word []byte, ds docStreams) map[int]paraFormat {
	_, _, fc, cb := fibOffsets(word)
	if cb == 0 || int(fc+cb) > len(ds.table) {
		return nil
	}
	plc := ds.table[fc : fc+cb]
	n := plcEntryCount(plc)
	if n <= 0 {
		return nil
	}
	fkpNums := plc[n*4 : n*4+n*4]
	paraMap := make(map[int]paraFormat)
	for i := 0; i < n; i++ {
		pn := int(binary.LittleEndian.Uint32(fkpNums[i*4 : i*4+4]))
		page := ds.readFKPPage(pn)
		if page == nil {
			continue
		}
		cpCount := int(page[511])
		if cpCount == 0 || cpCount > 64 {
			continue
		}
		for j := 0; j < cpCount; j++ {
			cpAtJ := int(binary.LittleEndian.Uint32(page[j*4 : j*4+4]))
			offsetWord := int(page[(cpCount+1)*4+j])
			if offsetWord == 0 {
				continue
			}
			byteOff := offsetWord * 2
			if byteOff >= 510 {
				continue
			}
			entryCb := int(page[byteOff])
			if entryCb == 0 {
				continue
			}
			if byteOff+1+entryCb*2 > 512 {
				entryCb = (511 - byteOff) / 2
			}
			if entryCb <= 0 {
				continue
			}
			rest := page[byteOff+1 : byteOff+1+entryCb*2]
			istd := int(binary.LittleEndian.Uint16(rest[:2]))
			_ = istd
			grpprl := rest[2:]
			if len(grpprl) > 1 {
				pf := decodeParaSprms(grpprl)
				paraMap[cpAtJ] = pf
			}
		}
	}
	return paraMap
}

func plcEntryCount(plc []byte) int {
	if len(plc) < 12 {
		return 0
	}
	return (len(plc) - 4) / 8
}

func decodeCharSprms(sprms []byte) charFormat {
	var f charFormat
	for i := 0; i+1 < len(sprms); i += 2 {
		sprmCode := binary.LittleEndian.Uint16(sprms[i : i+2])
		opSize := sprmOperandSize(sprmCode)
		opStart := i + 2
		if opSize == 0 || opStart+opSize > len(sprms) {
			continue
		}
		op := sprms[opStart : opStart+opSize]
		switch sprmCode {
		case sprmCBold:
			f.bold = op[0] != 0
		case sprmCItalic:
			f.italic = op[0] != 0
		case sprmCUnderline:
			f.underline = op[0]
		case sprmCStrike:
			f.strike = op[0] != 0
		case sprmCDoubleStrike:
			f.doubleStrike = op[0] != 0
		case sprmCAllCaps:
			f.allCaps = op[0] != 0
		case sprmCSmallCaps:
			f.smallCaps = op[0] != 0
		case sprmCHidden:
			f.hidden = op[0] != 0
		case sprmCSuperscript:
			f.superScript = op[0] != 0
			f.subScript = false
		case sprmCSubscript:
			f.subScript = op[0] != 0
			f.superScript = false
		case sprmCImprint:
			f.imprint = op[0] != 0
		case sprmCEmboss:
			f.emboss = op[0] != 0
		case sprmCFontSize:
			if len(op) >= 2 {
				f.fontSize = binary.LittleEndian.Uint16(op[:2])
			}
		case sprmCFontIndex:
			if len(op) >= 2 {
				f.fontIndex = binary.LittleEndian.Uint16(op[:2])
			}
		case sprmCColor:
			if len(op) >= 3 {
				f.fontColor = uint32(op[0]) | uint32(op[1])<<8 | uint32(op[2])<<16
			} else if len(op) >= 2 {
				f.fontColor = uint32(op[0]) | uint32(op[1])<<8
			}
		case sprmCHighlight:
			f.highlight = op[0]
		}
	}
	return f
}

func decodeParaSprms(sprms []byte) paraFormat {
	var f paraFormat
	for i := 0; i+1 < len(sprms); i += 2 {
		sprmCode := binary.LittleEndian.Uint16(sprms[i : i+2])
		opSize := sprmOperandSize(sprmCode)
		opStart := i + 2
		if opSize == 0 || opStart+opSize > len(sprms) {
			continue
		}
		op := sprms[opStart : opStart+opSize]
		switch sprmCode {
		case sprmPJc:
			f.alignment = op[0]
		case sprmPLeftIndent:
			if len(op) >= 2 {
				f.leftIndent = int16(binary.LittleEndian.Uint16(op[:2]))
			}
		case sprmPRightIndent:
			if len(op) >= 2 {
				f.rightIndent = int16(binary.LittleEndian.Uint16(op[:2]))
			}
		case sprmPFirstLine:
			if len(op) >= 2 {
				f.firstLine = int16(binary.LittleEndian.Uint16(op[:2]))
			}
		case sprmPSpaceBefore:
			if len(op) >= 2 {
				f.spaceBefore = binary.LittleEndian.Uint16(op[:2])
			}
		case sprmPSpaceAfter:
			if len(op) >= 2 {
				f.spaceAfter = binary.LittleEndian.Uint16(op[:2])
			}
		case sprmPLineSpacing:
			if len(op) >= 3 {
				f.lineRule = op[0]
				f.lineSpacing = int16(binary.LittleEndian.Uint16(op[1:3]))
			}
		case sprmPKeepTogether:
			f.keepTogether = op[0] != 0
		case sprmPKeepNext:
			f.keepNext = op[0] != 0
		case sprmPPageBreakBefore:
			f.pageBreakBefore = op[0] != 0
		case sprmPOutlineLevel:
			f.outlineLevel = op[0]
		case sprmPFInTable:
			f.inTable = op[0] != 0
		}
	}
	return f
}

func sprmOperandSize(opcode uint16) int {
	if opcode == sprmPFInTable {
		return 1
	}
	sprType := (opcode >> 13) & 0x07
	switch sprType {
	case 0:
		return 1
	case 1:
		return 2
	case 2:
		return 4
	case 3:
		return 0
	default:
		return 0
	}
}

func buildParaStyle(pf paraFormat) string {
	var parts []string
	switch pf.alignment {
	case 1:
		parts = append(parts, "text-align:center")
	case 2:
		parts = append(parts, "text-align:right")
	case 3:
		parts = append(parts, "text-align:justify")
	}
	if pf.leftIndent > 0 {
		parts = append(parts, "margin-left:"+itoa(int(pf.leftIndent)/20)+"pt")
	}
	if pf.rightIndent > 0 {
		parts = append(parts, "margin-right:"+itoa(int(pf.rightIndent)/20)+"pt")
	}
	if pf.firstLine != 0 {
		parts = append(parts, "text-indent:"+itoa(int(pf.firstLine)/20)+"pt")
	}
	if pf.spaceBefore > 0 {
		parts = append(parts, "margin-top:"+itoa(int(pf.spaceBefore)/20)+"pt")
	}
	if pf.spaceAfter > 0 {
		parts = append(parts, "margin-bottom:"+itoa(int(pf.spaceAfter)/20)+"pt")
	}
	if pf.lineSpacing != 0 && pf.lineRule != 0 {
		lineHeight := float64(pf.lineSpacing) / 240.0
		if pf.lineRule == 1 {
			lineHeight *= 1.2
		}
		if lineHeight > 0 && lineHeight < 5 {
			parts = append(parts, "line-height:"+itoaf(math.Round(lineHeight*10)/10))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ";")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func itoaf(f float64) string {
	tenth := int(f * 10)
	if tenth%10 == 0 {
		return itoa(tenth / 10)
	}
	return itoa(tenth/10) + "." + itoa(tenth%10)
}
