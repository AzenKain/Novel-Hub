package request

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"novelhub/pkg/constants"
)

func TestGetReadListsDto_GetLimit(t *testing.T) {
	t.Run("default limit when zero or negative", func(t *testing.T) {
		dto := GetReadListsDto{Limit: 0}
		assert.Equal(t, int64(50), dto.GetLimit())

		dtoNeg := GetReadListsDto{Limit: -10}
		assert.Equal(t, int64(50), dtoNeg.GetLimit())
	})

	t.Run("valid custom limit", func(t *testing.T) {
		dto := GetReadListsDto{Limit: 25}
		assert.Equal(t, int64(25), dto.GetLimit())
	})

	t.Run("cap limit when exceeding MaxPaginationLimit", func(t *testing.T) {
		dto := GetReadListsDto{Limit: constants.MaxPaginationLimit + 50}
		assert.Equal(t, int64(constants.MaxPaginationLimit), dto.GetLimit())
	})
}

func TestGetReadListsDto_ParseCursor(t *testing.T) {
	t.Run("empty cursor returns nil and empty id", func(t *testing.T) {
		dto := GetReadListsDto{Cursor: ""}
		tTime, id := dto.ParseCursor()
		assert.Nil(t, tTime)
		assert.Empty(t, id)
	})

	t.Run("cursor with timestamp and ID", func(t *testing.T) {
		now := time.Now().UTC()
		nowStr := now.Format(time.RFC3339Nano)
		dto := GetReadListsDto{Cursor: nowStr + "|list-uuid-123"}

		tTime, id := dto.ParseCursor()
		assert.NotNil(t, tTime)
		assert.Equal(t, "list-uuid-123", id)
		assert.Equal(t, now.Format(time.RFC3339Nano), tTime.Format(time.RFC3339Nano))
	})

	t.Run("cursor with timestamp only", func(t *testing.T) {
		now := time.Now().UTC()
		nowStr := now.Format(time.RFC3339Nano)
		dto := GetReadListsDto{Cursor: nowStr}

		tTime, id := dto.ParseCursor()
		assert.NotNil(t, tTime)
		assert.Empty(t, id)
	})

	t.Run("invalid cursor returns nil and empty id", func(t *testing.T) {
		dto := GetReadListsDto{Cursor: "invalid-timestamp|list-uuid"}
		tTime, id := dto.ParseCursor()
		assert.Nil(t, tTime)
		assert.Empty(t, id)
	})
}
