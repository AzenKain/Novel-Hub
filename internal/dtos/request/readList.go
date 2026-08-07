package request

import (
	"strings"
	"time"

	"novelhub/pkg/constants"
)

type GetReadListsDto struct {
	Limit  int64  `json:"limit,omitempty" query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string `json:"cursor,omitempty" query:"cursor" validate:"omitempty,readlist_cursor"`
}

func (d *GetReadListsDto) GetLimit() int64 {
	if d.Limit <= 0 {
		return 50
	}
	if d.Limit > constants.MaxPaginationLimit {
		return constants.MaxPaginationLimit
	}
	return d.Limit
}

func (d *GetReadListsDto) ParseCursor() (*time.Time, string) {
	if d.Cursor == "" {
		return nil, ""
	}
	parts := strings.SplitN(d.Cursor, "|", 2)
	if len(parts) == 2 {
		if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
			return &t, parts[1]
		}
	} else if t, err := time.Parse(time.RFC3339Nano, d.Cursor); err == nil {
		return &t, ""
	}
	return nil, ""
}

type CreateReadListDto struct {
	Name        string `json:"name" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"max=2000"`
}

type UpdateReadListDto struct {
	Name        string `json:"name" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"max=2000"`
}

type AddReadListBookDto struct {
	BookID string `json:"book_id" validate:"required"`
}

type ReorderReadListDto struct {
	BookIDs []string `json:"book_ids" validate:"required,min=1,max=2000,dive,required"`
}

type NextInOrderQueryDto struct {
	After string `json:"after,omitempty" query:"after" validate:"omitempty"`
}
