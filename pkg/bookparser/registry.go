package bookparser

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Registry interface {
	Register(parser Parser, formats ...string)
	ParserForFormat(format string) (Parser, bool)
	ParserForPath(path string) (Parser, bool)
	Parser(format string, path string) (Parser, error)
	HasFormat(format string) bool
	HasPath(path string) bool
}

type parserRegistry struct {
	parsers map[string]Parser
}

func NewRegistry() Registry {
	return &parserRegistry{parsers: make(map[string]Parser)}
}

func (r *parserRegistry) Register(parser Parser, formats ...string) {
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

func (r *parserRegistry) ParserForFormat(format string) (Parser, bool) {
	if r == nil {
		return nil, false
	}
	parser, ok := r.parsers[NormalizeFormat(format)]
	return parser, ok
}

func (r *parserRegistry) ParserForPath(path string) (Parser, bool) {
	return r.ParserForFormat(FormatFromPath(path))
}

func (r *parserRegistry) Parser(format string, path string) (Parser, error) {
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

func (r *parserRegistry) HasFormat(format string) bool {
	_, ok := r.ParserForFormat(format)
	return ok
}

func (r *parserRegistry) HasPath(path string) bool {
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
