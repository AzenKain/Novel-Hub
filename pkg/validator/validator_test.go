package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type sampleQuery struct {
	Cursor string `query:"cursor" validate:"omitempty,readlist_cursor"`
}

func TestReadListCursorValidation(t *testing.T) {
	tests := []struct {
		name    string
		cursor  string
		wantErr bool
	}{
		{
			name:    "empty cursor is valid",
			cursor:  "",
			wantErr: false,
		},
		{
			name:    "RFC3339 timestamp only is valid",
			cursor:  time.Now().Format(time.RFC3339Nano),
			wantErr: false,
		},
		{
			name:    "RFC3339 timestamp with pipe and ID is valid",
			cursor:  time.Now().Format(time.RFC3339Nano) + "|readlist-123",
			wantErr: false,
		},
		{
			name:    "invalid timestamp format returns error",
			cursor:  "invalid-date|123",
			wantErr: true,
		},
		{
			name:    "random garbage string returns error",
			cursor:  "random_garbage",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sampleQuery{Cursor: tt.cursor}
			err := validate.Struct(s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type sampleServerURL struct {
	ServerURL *string `json:"server.url" validate:"omitempty,server_url,max=2048"`
}

func TestServerURLValidation(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name      string
		serverURL *string
		wantErr   bool
	}{
		{
			name:      "nil pointer is valid",
			serverURL: nil,
			wantErr:   false,
		},
		{
			name:      "empty string pointer is valid",
			serverURL: strPtr(""),
			wantErr:   false,
		},
		{
			name:      "whitespace only string pointer is valid",
			serverURL: strPtr("   "),
			wantErr:   false,
		},
		{
			name:      "valid http localhost is valid",
			serverURL: strPtr("http://localhost:8080"),
			wantErr:   false,
		},
		{
			name:      "valid https domain is valid",
			serverURL: strPtr("https://novelhub.example.com"),
			wantErr:   false,
		},
		{
			name:      "valid https with trailing slash is valid",
			serverURL: strPtr("https://novelhub.example.com/"),
			wantErr:   false,
		},
		{
			name:      "invalid URL with path returns error",
			serverURL: strPtr("https://novelhub.example.com/books"),
			wantErr:   true,
		},
		{
			name:      "invalid URL with query returns error",
			serverURL: strPtr("https://novelhub.example.com?query=1"),
			wantErr:   true,
		},
		{
			name:      "invalid scheme returns error",
			serverURL: strPtr("ftp://example.com"),
			wantErr:   true,
		},
		{
			name:      "newlines in URL returns error",
			serverURL: strPtr("https://example.com\r\nX-Bad: 1"),
			wantErr:   true,
		},
		{
			name:      "garbage text returns error",
			serverURL: strPtr("not-a-valid-url"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sampleServerURL{ServerURL: tt.serverURL}
			err := validate.Struct(s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	str := "invalid_url"
	s := sampleServerURL{ServerURL: &str}
	err := validate.Struct(s)
	assert.Error(t, err)

	errList := formatValidationError(err)
	assert.Len(t, errList, 1)
	assert.Equal(t, "server.url", errList[0].FailedField)
	assert.Equal(t, "server_url", errList[0].Tag)
	assert.Equal(t, "server.url must be a valid http or https URL", errList[0].Message)
}
