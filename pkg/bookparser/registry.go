package bookparser

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Registry struct {
	parsers map[string]Parser
}

func NewRegistry() *Registry {
	return &Registry{parsers: make(map[string]Parser)}
}

func (r *Registry) Register(parser Parser, formats ...string) {
	if r == nil || parser == nil {
		return
	}
	for _, format := range formats {
		key := NormalizeFormat(format)
		if key == "" {
			continue
		}
		r.parsers[key] = parser
	}
}

func (r *Registry) ParserForFormat(format string) (Parser, bool) {
	if r == nil {
		return nil, false
	}
	parser, ok := r.parsers[NormalizeFormat(format)]
	return parser, ok
}

func (r *Registry) ParserForPath(path string) (Parser, bool) {
	return r.ParserForFormat(FormatFromPath(path))
}

func (r *Registry) Parser(format string, path string) (Parser, error) {
	if parser, ok := r.ParserForFormat(format); ok {
		return parser, nil
	}
	if parser, ok := r.ParserForPath(path); ok {
		return parser, nil
	}
	name := NormalizeFormat(format)
	if name == "" {
		name = FormatFromPath(path)
	}
	return nil, fmt.Errorf("no reader parser registered for format %q", name)
}

func (r *Registry) HasFormat(format string) bool {
	_, ok := r.ParserForFormat(format)
	return ok
}

func (r *Registry) HasPath(path string) bool {
	_, ok := r.ParserForPath(path)
	return ok
}

func NormalizeFormat(format string) string {
	format = strings.TrimSpace(strings.ToLower(format))
	format = strings.TrimPrefix(format, ".")
	if format == "markdown" {
		return "md"
	}
	if format == "jpeg" {
		return "jpg"
	}
	return format
}

func FormatFromPath(path string) string {
	name := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(name, ".kepub.epub") {
		return "kepub.epub"
	}
	return NormalizeFormat(filepath.Ext(name))
}
