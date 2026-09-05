// Package flacn demuxes native FLAC framing (RFC 9639): the fLaC marker, metadata blocks, and the self-framing audio stream.
package flacn

// Match reports whether head begins with the fLaC stream marker.
func Match(head []byte) bool {
	return len(head) >= 4 && string(head[:4]) == "fLaC"
}
