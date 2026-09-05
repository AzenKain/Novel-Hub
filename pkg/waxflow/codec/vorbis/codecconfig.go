package vorbis

// PackHeaders serializes the three Vorbis header packets into one codec-config blob (Xiph lacing).
func PackHeaders(id, comment, setup []byte) []byte {
	out := []byte{2}
	out = appendXiphLen(out, len(id))
	out = appendXiphLen(out, len(comment))
	out = append(out, id...)
	out = append(out, comment...)
	out = append(out, setup...)
	return out
}

func appendXiphLen(out []byte, n int) []byte {
	for n >= 255 {
		out = append(out, 255)
		n -= 255
	}
	return append(out, byte(n))
}

// SplitConfig reverses PackHeaders, returning the identification, comment, and setup packets.
func SplitConfig(blob []byte) (id, comment, setup []byte, err error) {
	return splitHeaders(blob)
}

func splitHeaders(blob []byte) (id, comment, setup []byte, err error) {
	if len(blob) < 1 || blob[0] != 2 {
		return nil, nil, nil, malformed("codec config: want 3 Vorbis headers, got count byte %d", firstByte(blob))
	}
	pos := 1
	lenID, pos, err := readXiphLen(blob, pos)
	if err != nil {
		return nil, nil, nil, err
	}
	lenComment, pos, err := readXiphLen(blob, pos)
	if err != nil {
		return nil, nil, nil, err
	}
	if pos+lenID+lenComment > len(blob) {
		return nil, nil, nil, malformed("codec config: header lengths exceed blob")
	}
	id = blob[pos : pos+lenID]
	comment = blob[pos+lenID : pos+lenID+lenComment]
	setup = blob[pos+lenID+lenComment:]
	return id, comment, setup, nil
}

func readXiphLen(blob []byte, pos int) (n, next int, err error) {
	for {
		if pos >= len(blob) {
			return 0, 0, malformed("codec config: truncated length")
		}
		b := blob[pos]
		pos++
		n += int(b)
		if b < 255 {
			return n, pos, nil
		}
	}
}

func firstByte(b []byte) int {
	if len(b) == 0 {
		return -1
	}
	return int(b[0])
}

// ParseConfig unpacks a codec-config blob and parses it into a Config.
func ParseConfig(blob []byte) (Config, error) {
	id, comment, setup, err := splitHeaders(blob)
	if err != nil {
		return Config{}, err
	}
	return ParseHeaders(id, comment, setup)
}
