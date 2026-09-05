// Package id3 parses the byte length of a leading ID3v2 tag.
package id3

// Size returns the total byte length of an ID3v2 tag starting at b, or 0 when b does not begin with one.
func Size(b []byte) int64 {
	if len(b) < 10 || string(b[:3]) != "ID3" {
		return 0
	}
	for _, x := range b[6:10] {
		if x&0x80 != 0 {
			return 0
		}
	}
	n := int64(b[6])<<21 | int64(b[7])<<14 | int64(b[8])<<7 | int64(b[9])
	n += 10
	if b[5]&0x10 != 0 {
		n += 10
	}
	return n
}
