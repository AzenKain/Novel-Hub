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
