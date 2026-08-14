package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

type SeriesIndexMatch struct {
	SeriesKey   string
	SeriesIndex string
	BookID      string
}

type ReadListRepository interface {
	CreateReadList(ctx context.Context, id, userID, name, description string) (*models.ReadListEntity, error)
	UpdateReadList(ctx context.Context, id, userID, name, description string) (*models.ReadListEntity, error)
	DeleteReadList(ctx context.Context, id, userID string) error
	ReadListOwnedByUser(ctx context.Context, readListID, userID string) (bool, error)
	GetUserReadLists(ctx context.Context, userID string, cursorCreatedAt *time.Time, cursorID string, limit int64) ([]*models.ReadListEntity, error)
	GetReadListsByIDs(ctx context.Context, ids []string) ([]*models.ReadListEntity, error)
	GetReadListBookIDs(ctx context.Context, readListID string) ([]string, error)
	CountBooksInReadLists(ctx context.Context, readListIDs []string) (map[string]int64, error)
	AppendBookToReadList(ctx context.Context, readListID, bookID string) error
	RemoveBookFromReadList(ctx context.Context, readListID, bookID string) error
	ReplaceReadListOrder(ctx context.Context, readListID string, bookIDs []string) error
	GetNextInReadList(ctx context.Context, readListID, afterBookID string) (string, error)
	GetFirstInReadList(ctx context.Context, readListID string) (string, error)
	MatchBooksBySeriesNames(ctx context.Context, seriesKeys []string) ([]SeriesIndexMatch, error)
	InvalidateReadListCache(ctx context.Context, readListID, userID string)
	WithTx(tx *sql.Tx) ReadListRepository
}

type readListRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
	c       cache.Cache
	inTx    bool
	sfg     *singleflight.Group
}

func NewReadListRepository(db *sql.DB, c cache.Cache) ReadListRepository {
	return &readListRepository{
		db:      db,
		queries: sqlc.New(db),
		c:       c,
		sfg:     &singleflight.Group{},
	}
}

// Both halves matter: the transactional copy needs its own singleflight group AND inTx set.
// Sharing the parent's group lets a plain reader join a call already in flight inside the
// transaction and take back a row that is not committed yet — one that vanishes on rollback
// while the cache keeps serving it.
func (r *readListRepository) WithTx(tx *sql.Tx) ReadListRepository {
	if tx == nil {
		return r
	}
	return &readListRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
		c:       r.c,
		inTx:    true,
		sfg:     &singleflight.Group{},
	}
}

// InvalidateReadListCache drops every cached view of one list. Call it after tx.Commit() for
// mutations made through WithTx, whose own invalidation is deferred for the reason above.
// Pass "" for userID when the owner is not at hand — the list-scoped entries still go.
func (r *readListRepository) InvalidateReadListCache(ctx context.Context, readListID, userID string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("read_list", "items", readListID), cache.BuildKey("read_list", "id", readListID))
	_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListCountsPattern)
	if userID != "" {
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("read_list", "user", userID)+"*")
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListOwnedPattern)
	}
}

func (r *readListRepository) CreateReadList(ctx context.Context, id, userID, name, description string) (*models.ReadListEntity, error) {
	row, err := r.queries.CreateReadList(ctx, sqlc.CreateReadListParams{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Description: convert.StrPtrToNullStringNonEmpty(&description),
	})
	if err != nil {
		return nil, err
	}
	result := (&models.ReadListEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("read_list", "user", userID)+"*")
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListOwnedPattern)
		_ = r.c.Set(ctx, cache.BuildKey("read_list", "id", result.ID), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *readListRepository) UpdateReadList(ctx context.Context, id, userID, name, description string) (*models.ReadListEntity, error) {
	row, err := r.queries.UpdateReadList(ctx, sqlc.UpdateReadListParams{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Description: convert.StrPtrToNullStringNonEmpty(&description),
	})
	if err != nil {
		return nil, err
	}
	result := (&models.ReadListEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("read_list", "user", userID)+"*")
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListOwnedPattern)
		_ = r.c.Set(ctx, cache.BuildKey("read_list", "id", result.ID), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *readListRepository) DeleteReadList(ctx context.Context, id, userID string) error {
	if err := r.queries.DeleteReadList(ctx, sqlc.DeleteReadListParams{ID: id, UserID: userID}); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("read_list", "user", userID)+"*")
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListOwnedPattern)
		_ = r.c.Del(ctx, cache.BuildKey("read_list", "items", id))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListCountsPattern)
		_ = r.c.Del(ctx, cache.BuildKey("read_list", "id", id))
	}
	return nil
}

func (r *readListRepository) ReadListOwnedByUser(ctx context.Context, readListID, userID string) (bool, error) {
	key := cache.BuildKey("read_list", "owned", userID, readListID)
	if r.c != nil && !r.inTx {
		var owned bool
		if err := r.c.Get(ctx, key, &owned); err == nil {
			return owned, nil
		}
	}

	value, err, _ := r.sfg.Do(key, func() (any, error) {
		owned, err := r.queries.ReadListOwnedByUser(ctx, sqlc.ReadListOwnedByUserParams{
			ID:     readListID,
			UserID: userID,
		})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, owned, constants.NormalCacheDuration)
		}
		return owned, nil
	})
	if err != nil {
		return false, err
	}
	return value.(bool), nil
}

func (r *readListRepository) GetUserReadLists(ctx context.Context, userID string, cursorCreatedAt *time.Time, cursorID string, limit int64) ([]*models.ReadListEntity, error) {
	var key string
	if cursorCreatedAt == nil {
		key = cache.BuildKey("read_list", "user", userID, "limit", limit)
	}

	if key != "" && r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetReadListsByIDs(ctx, ids)
		}
	}

	sfKey := key
	if sfKey == "" {
		sfKey = fmt.Sprintf("read_list:user:%s:cursor:%v:%s:limit:%d", userID, cursorCreatedAt, cursorID, limit)
	}

	v, err, _ := r.sfg.Do(sfKey, func() (any, error) {
		ids, err := r.queries.GetUserReadListIDs(ctx, sqlc.GetUserReadListIDsParams{
			UserID:          userID,
			CursorCreatedAt: cursorTimeArg(cursorCreatedAt),
			CursorID:        convert.StrPtrToNullString(&cursorID),
			Limit:           limit,
		})
		if err != nil {
			return nil, err
		}
		if key != "" && r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetReadListsByIDs(ctx, v.([]string))
}

func (r *readListRepository) GetReadListsByIDs(ctx context.Context, ids []string) ([]*models.ReadListEntity, error) {
	if len(ids) == 0 {
		return []*models.ReadListEntity{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("read_list", "id", id)
	}

	byID := make(map[string]*models.ReadListEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))
	missingKeys := make([]string, 0, len(ids))

	if r.c != nil && !r.inTx {
		raws := r.c.MGet(ctx, keys...)
		for i, raw := range raws {
			if len(raw) > 0 {
				var entity models.ReadListEntity
				if err := jsonx.Unmarshal(raw, &entity); err == nil {
					byID[entity.ID] = &entity
					continue
				}
			}
			missingIDs = append(missingIDs, ids[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIDs = ids
		missingKeys = keys
	}

	if len(missingIDs) > 0 {
		rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.ReadList, error) {
			return r.queries.GetReadListsByIDs(ctx, chunk)
		})
		if err != nil {
			return nil, err
		}
		counts, _ := r.CountBooksInReadLists(ctx, missingIDs)
		fetched := make(map[string]*models.ReadListEntity, len(rows))
		for _, row := range rows {
			entity := (&models.ReadListEntity{}).FromSqlc(row)
			if cnt, ok := counts[entity.ID]; ok {
				entity.BookCount = cnt
			}
			byID[entity.ID] = entity
			fetched[entity.ID] = entity
		}

		if r.c != nil && !r.inTx {
			toCache := make(map[string]any, len(fetched))
			for i, missingID := range missingIDs {
				if entity, ok := fetched[missingID]; ok {
					toCache[missingKeys[i]] = entity
				}
			}
			if len(toCache) > 0 {
				_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
			}
		}
	}

	ordered := make([]*models.ReadListEntity, 0, len(ids))
	for _, id := range ids {
		if entity, ok := byID[id]; ok {
			ordered = append(ordered, entity)
		}
	}
	return ordered, nil
}

func (r *readListRepository) GetReadListBookIDs(ctx context.Context, readListID string) ([]string, error) {
	key := cache.BuildKey("read_list", "items", readListID)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return ids, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		ids, err := r.queries.GetReadListBookIDs(ctx, readListID)
		if err != nil {
			return nil, err
		}
		if ids == nil {
			ids = []string{}
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

// One grouped count for the whole page instead of one COUNT per list — the list view renders a
// book count on every card, and the per-list version was a query per row.
func (r *readListRepository) CountBooksInReadLists(ctx context.Context, readListIDs []string) (map[string]int64, error) {
	counts := make(map[string]int64, len(readListIDs))
	if len(readListIDs) == 0 {
		return counts, nil
	}
	key := cache.BuildKey("read_list", "counts", strings.Join(readListIDs, ","))
	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, key, &counts); err == nil {
			return counts, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		payload, err := jsonx.MarshalString(readListIDs)
		if err != nil {
			return nil, err
		}
		rows, err := r.queries.CountBooksInReadLists(ctx, payload)
		if err != nil {
			return nil, err
		}
		fresh := make(map[string]int64, len(readListIDs))
		for _, id := range readListIDs {
			fresh[id] = 0
		}
		for _, row := range rows {
			fresh[row.ReadListID] = row.BookCount
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, fresh, constants.ListCacheDuration)
		}
		return fresh, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(map[string]int64), nil
}

func (r *readListRepository) AppendBookToReadList(ctx context.Context, readListID, bookID string) error {
	if err := r.queries.AppendBookToReadList(ctx, sqlc.AppendBookToReadListParams{
		ReadListID: readListID,
		BookID:     bookID,
	}); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("read_list", "items", readListID))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListCountsPattern)
	}
	return nil
}

func (r *readListRepository) RemoveBookFromReadList(ctx context.Context, readListID, bookID string) error {
	if err := r.queries.RemoveBookFromReadList(ctx, sqlc.RemoveBookFromReadListParams{
		ReadListID: readListID,
		BookID:     bookID,
	}); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("read_list", "items", readListID))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListCountsPattern)
	}
	return nil
}

func (r *readListRepository) ReplaceReadListOrder(ctx context.Context, readListID string, bookIDs []string) error {
	for i, bookID := range bookIDs {
		if err := r.queries.SetReadListBookPosition(ctx, sqlc.SetReadListBookPositionParams{
			Position:   int64(i),
			ReadListID: readListID,
			BookID:     bookID,
		}); err != nil {
			return err
		}
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("read_list", "items", readListID))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyReadListCountsPattern)
	}
	return nil
}

// Only the id comes back: the stored position has gaps after a removal, so it disagrees with the
// index the list view renders and callers derive the display position from the ordered id list.
func (r *readListRepository) GetNextInReadList(ctx context.Context, readListID, afterBookID string) (string, error) {
	row, err := r.queries.GetNextInReadList(ctx, sqlc.GetNextInReadListParams{
		ReadListID:  readListID,
		AfterBookID: afterBookID,
	})
	if err != nil {
		return "", err
	}
	return row.BookID, nil
}

func (r *readListRepository) GetFirstInReadList(ctx context.Context, readListID string) (string, error) {
	row, err := r.queries.GetFirstInReadList(ctx, readListID)
	if err != nil {
		return "", err
	}
	return row.BookID, nil
}

func (r *readListRepository) MatchBooksBySeriesNames(ctx context.Context, seriesKeys []string) ([]SeriesIndexMatch, error) {
	if len(seriesKeys) == 0 {
		return nil, nil
	}
	payload, err := jsonx.MarshalString(seriesKeys)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.MatchBooksBySeriesNames(ctx, payload)
	if err != nil {
		return nil, err
	}
	out := make([]SeriesIndexMatch, 0, len(rows))
	for _, row := range rows {
		out = append(out, SeriesIndexMatch{
			SeriesKey:   row.SeriesKey,
			SeriesIndex: row.SeriesIndex.String,
			BookID:      row.BookID,
		})
	}
	return out, nil
}
