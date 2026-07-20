package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type SeriesEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *SeriesEntity) FromSqlc(res sqlc.Series) *SeriesEntity {
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	return e
}

type SeriesEntities []*SeriesEntity

func (e *SeriesEntities) FromSqlc(rows []sqlc.Series) []*SeriesEntity {
	slice := make([]*SeriesEntity, len(rows))
	flat := make([]SeriesEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

type PublisherEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *PublisherEntity) FromSqlc(res sqlc.Publisher) *PublisherEntity {
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	return e
}

type PublisherEntities []*PublisherEntity

func (e *PublisherEntities) FromSqlc(rows []sqlc.Publisher) []*PublisherEntity {
	slice := make([]*PublisherEntity, len(rows))
	flat := make([]PublisherEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

type LanguageEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *LanguageEntity) FromSqlc(res sqlc.Language) *LanguageEntity {
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	return e
}

type LanguageEntities []*LanguageEntity

func (e *LanguageEntities) FromSqlc(rows []sqlc.Language) []*LanguageEntity {
	slice := make([]*LanguageEntity, len(rows))
	flat := make([]LanguageEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

type MetadataCountEntity struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BookCount int64  `json:"bookCount"`
	CoverURL  string `json:"coverUrl,omitempty"`
}

type AuthorEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Bio       *string   `json:"bio"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (e *AuthorEntity) FromSqlc(res sqlc.Author) *AuthorEntity {
	e.ID = res.ID
	e.Name = res.Name
	e.Bio = convert.NullStringToStrPtr(res.Bio)
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type AuthorEntities []*AuthorEntity

func (e *AuthorEntities) FromSqlc(rows []sqlc.Author) []*AuthorEntity {
	slice := make([]*AuthorEntity, len(rows))
	flat := make([]AuthorEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

type TagEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *TagEntity) FromSqlc(res sqlc.Tag) *TagEntity {
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	return e
}

type TagEntities []*TagEntity

func (e *TagEntities) FromSqlc(rows []sqlc.Tag) []*TagEntity {
	slice := make([]*TagEntity, len(rows))
	flat := make([]TagEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}
