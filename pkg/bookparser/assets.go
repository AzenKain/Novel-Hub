package bookparser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
)

type EmbeddedImageAsset struct {
	Name string
	Data []byte
}

func ExtractEmbeddedImageAssets(data []byte) []EmbeddedImageAsset {
	return AppendEmbeddedImageAssets(nil, data)
}

func AppendEmbeddedImageAssets(assets []EmbeddedImageAsset, data []byte) []EmbeddedImageAsset {
	for offset := 0; offset < len(data); {
		image, ext, ok := embeddedImageAt(data, offset)
		if !ok {
			offset++
			continue
		}
		assets = append(assets, EmbeddedImageAsset{
			Name: fmt.Sprintf("images/embedded-%04d.%s", len(assets)+1, ext),
			Data: image,
		})
		offset += len(image)
	}
	return assets
}

func EmbeddedImageNames(assets []EmbeddedImageAsset) []string {
	images := make([]string, 0, len(assets))
	for _, asset := range assets {
		images = append(images, asset.Name)
	}
	return images
}

func FindEmbeddedImageAsset(assets []EmbeddedImageAsset, assetPath string) ([]byte, error) {
	assetPath, _, _ = strings.Cut(assetPath, "?")
	assetPath, _, _ = strings.Cut(assetPath, "#")
	assetPath = strings.ToLower(strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(assetPath)), "/"))
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if assetPath == name || assetPath == strings.TrimPrefix(name, "images/") {
			return asset.Data, nil
		}
	}
	return nil, fmt.Errorf("embedded image asset %q not found", assetPath)
}

func embeddedImageAt(data []byte, offset int) ([]byte, string, bool) {
	switch {
	case hasPrefixAt(data, offset, []byte{0xff, 0xd8, 0xff}):
		if end := jpegEnd(data, offset); end > offset {
			return data[offset:end], "jpg", true
		}
	case hasPrefixAt(data, offset, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		if end := pngEnd(data, offset); end > offset {
			return data[offset:end], "png", true
		}
	case hasPrefixAt(data, offset, []byte("GIF87a")) || hasPrefixAt(data, offset, []byte("GIF89a")):
		if end := gifEnd(data, offset); end > offset {
			return data[offset:end], "gif", true
		}
	case hasPrefixAt(data, offset, []byte("RIFF")) && len(data) >= offset+12 && bytes.Equal(data[offset+8:offset+12], []byte("WEBP")):
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		end := offset + 8 + size
		if size > 4 && end <= len(data) {
			return data[offset:end], "webp", true
		}
	case hasPrefixAt(data, offset, []byte("BM")) && len(data) >= offset+6:
		size := int(binary.LittleEndian.Uint32(data[offset+2 : offset+6]))
		end := offset + size
		if size >= 14 && end <= len(data) {
			return data[offset:end], "bmp", true
		}
	}
	return nil, "", false
}

func hasPrefixAt(data []byte, offset int, prefix []byte) bool {
	return offset >= 0 && offset+len(prefix) <= len(data) && bytes.Equal(data[offset:offset+len(prefix)], prefix)
}

func jpegEnd(data []byte, start int) int {
	for i := start + 3; i+1 < len(data); i++ {
		if data[i] == 0xff && data[i+1] == 0xd9 {
			return i + 2
		}
	}
	return -1
}

func pngEnd(data []byte, start int) int {
	iend := bytes.Index(data[start:], []byte("IEND"))
	if iend < 0 {
		return -1
	}
	chunkType := start + iend
	end := chunkType + 8
	if chunkType >= start+12 && end <= len(data) {
		return end
	}
	return -1
}

func gifEnd(data []byte, start int) int {
	for i := start + 13; i < len(data); i++ {
		if data[i] == 0x3b {
			return i + 1
		}
	}
	return -1
}
