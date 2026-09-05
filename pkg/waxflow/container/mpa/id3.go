package mpa

import "novelhub/pkg/waxflow/container"

var id3Text = []struct{ key, frame string }{
	{"TITLE", "TIT2"},
	{"ARTIST", "TPE1"},
	{"ALBUM", "TALB"},
	{"ALBUMARTIST", "TPE2"},
	{"COMPOSER", "TCOM"},
	{"GENRE", "TCON"},
	{"RECORDINGDATE", "TDRC"},
}

const maxID3Bytes = 48 << 10

func id3v2Tag(tags []container.Tag) []byte {
	vals := make(map[string][]string, len(tags))
	for _, t := range tags {
		if t.Value != "" {
			vals[t.Key] = append(vals[t.Key], t.Value)
		}
	}
	var frames []byte
	add := func(id, text string) {
		if text == "" || len(frames)+10+1+len(text) > maxID3Bytes {
			return
		}
		frames = append(frames, id...)
		frames = appendSyncsafe(frames, uint32(1+len(text)))
		frames = append(frames, 0, 0)
		frames = append(frames, 3)
		frames = append(frames, text...)
	}
	for _, m := range id3Text {
		if vs := vals[m.key]; len(vs) > 0 {
			add(m.frame, joinValues(vs))
		}
	}
	add("TRCK", numberPair(vals["TRACKNUMBER"], vals["TRACKTOTAL"]))
	add("TPOS", numberPair(vals["DISCNUMBER"], vals["DISCTOTAL"]))
	if len(frames) == 0 {
		return nil
	}
	tag := make([]byte, 0, 10+len(frames))
	tag = append(tag, "ID3"...)
	tag = append(tag, 4, 0, 0)
	tag = appendSyncsafe(tag, uint32(len(frames)))
	return append(tag, frames...)
}

func appendSyncsafe(b []byte, v uint32) []byte {
	return append(b, byte(v>>21&0x7F), byte(v>>14&0x7F), byte(v>>7&0x7F), byte(v&0x7F))
}

func joinValues(vs []string) string {
	out := vs[0]
	for _, v := range vs[1:] {
		out += "; " + v
	}
	return out
}

func numberPair(nums, totals []string) string {
	n := ""
	if len(nums) > 0 {
		n = nums[0]
	}
	if len(totals) > 0 && totals[0] != "" {
		if n == "" {
			n = "0"
		}
		return n + "/" + totals[0]
	}
	return n
}
