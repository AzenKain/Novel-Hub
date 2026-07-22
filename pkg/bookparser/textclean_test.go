package bookparser_test

import (
	"testing"

	"novelhub/pkg/bookparser"
)

func TestCleanChapterTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal title",
			input:    "Chapter 1",
			expected: "Chapter 1",
		},
		{
			name:     "URL encoded Vietnamese title",
			input:    "M%E1%BB%A5c%20L%E1%BB%A5c",
			expected: "Mục Lục",
		},
		{
			name:     "URL encoded title with spaces and symbols",
			input:    "Ch%C6%B0%C6%A1ng%201%20%E2%80%93%20Murasaki-san",
			expected: "Chương 1 – Murasaki-san",
		},
		{
			name:     "HTML entity encoded title",
			input:    "Tom &amp; Jerry &quot;Special&quot;",
			expected: "Tom & Jerry \"Special\"",
		},
		{
			name:     "Combined URL and HTML encoding",
			input:    "Ch%C6%B0%C6%A1ng%20k%E1%BA%BFt%20&amp;%20L%E1%BB%9Di%20b%E1%BA%A1t",
			expected: "Chương kết & Lời bạt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bookparser.CleanChapterTitle(tt.input)
			if result != tt.expected {
				t.Errorf("CleanChapterTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
