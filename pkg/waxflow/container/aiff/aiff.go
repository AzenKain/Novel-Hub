// Package aiff reads and writes AIFF and AIFF-C, the Apple/SGI audio container.
package aiff

import (
	"encoding/binary"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec/pcm"
	"novelhub/pkg/waxflow/waxerr"
)

const (
	idFORM = "FORM"
	idAIFF = "AIFF"
	idAIFC = "AIFC"
	idCOMM = "COMM"
	idSSND = "SSND"
	idFVER = "FVER"
)

const (
	compNONE = "NONE"
	compTwos = "twos"
	compSowt = "sowt"
	compRaw  = "raw "
	compFl32 = "fl32"
	compFL32 = "FL32"
	compFl64 = "fl64"
	compFL64 = "FL64"
)

const fverTimestamp = 0xA2805140

const size32Max = 0xFFFFFFFF

// Match reports whether head (at least 12 bytes) looks like an AIFF or AIFF-C file.
func Match(head []byte) bool {
	if len(head) < 12 {
		return false
	}
	form := string(head[8:12])
	return string(head[:4]) == idFORM && (form == idAIFF || form == idAIFC)
}

// DefaultConfig returns the natural AIFF wire encoding for a pipeline format: big-endian signed integers packed in whole bytes (plain AIFF), float32 for the float domain (AIFF-C fl32).
func DefaultConfig(f audio.Format) (pcm.Config, error) {
	if err := f.Valid(); err != nil {
		return pcm.Config{}, err
	}
	if f.Type == audio.Float {
		return pcm.Config{Encoding: pcm.Float, Bits: 32, BigEndian: true}, nil
	}
	bits := pcm.ContainerBits(f.BitDepth)
	cfg := pcm.Config{Encoding: pcm.SignedInt, Bits: bits, BigEndian: bits > 8}
	if f.BitDepth != bits {
		cfg.ValidBits = f.BitDepth
	}
	if err := cfg.Validate(); err != nil {
		return pcm.Config{}, waxerr.Wrap(waxerr.CodeUnsupportedFormat, "aiff: no aiff encoding for format", err)
	}
	return cfg, nil
}

var be = binary.BigEndian
