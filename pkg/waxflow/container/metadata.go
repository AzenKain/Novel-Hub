package container

import "time"

// Tag is one canonical metadata field for muxers that can embed tags in their stream form.
type Tag struct {
	Key   string
	Value string
}

// Chapter is one chapter marker for muxers (and demuxers) that carry chapters.
type Chapter struct {
	Start time.Duration
	End   time.Duration
	Title string
}

// Chapterer is implemented by demuxers that parse chapter markers, in the same idiom as Warner and Indexer: an honest capability gate rather than a method every demuxer carries, since a container with no chapter form has nothing to answer and does not implement it.
type Chapterer interface {
	Chapters() []Chapter
}

// Tagger is implemented by demuxers that parse embedded tags, the same capability gate as Chapterer and for the same reason: a container with no tag form has nothing to answer and does not implement it.
type Tagger interface {
	Tags() map[string][]string
}

// Picture is embedded cover art for muxers that can embed it.
type Picture struct {
	MIME string
	Data []byte
}

// ValidTagKey reports whether key is legal as a Vorbis comment field name: printable ASCII 0x20 to 0x7D excluding '=', non-empty.
func ValidTagKey(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		if c := key[i]; c < 0x20 || c > 0x7D || c == '=' {
			return false
		}
	}
	return true
}
