package models

import (
	"time"

	"novelhub/internal/dtos/response"
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

func (e *SeriesEntity) ToResponse() *response.SeriesResponse {
	if e == nil {
		return nil
	}
	return &response.SeriesResponse{
		ID:        e.ID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
	}
}

func SeriesEntitiesToResponse(entities []*SeriesEntity) []*response.SeriesResponse {
	out := make([]*response.SeriesResponse, 0, len(entities))
	for _, s := range entities {
		if s == nil {
			continue
		}
		out = append(out, s.ToResponse())
	}
	return out
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

func (e *PublisherEntity) ToResponse() *response.PublisherResponse {
	if e == nil {
		return nil
	}
	return &response.PublisherResponse{
		ID:        e.ID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
	}
}

func PublisherEntitiesToResponse(entities []*PublisherEntity) []*response.PublisherResponse {
	out := make([]*response.PublisherResponse, 0, len(entities))
	for _, p := range entities {
		if p == nil {
			continue
		}
		out = append(out, p.ToResponse())
	}
	return out
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

func (e *LanguageEntity) ToResponse() *response.LanguageResponse {
	if e == nil {
		return nil
	}
	return &response.LanguageResponse{
		ID:        e.ID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
	}
}

func LanguageEntitiesToResponse(entities []*LanguageEntity) []*response.LanguageResponse {
	out := make([]*response.LanguageResponse, 0, len(entities))
	for _, l := range entities {
		if l == nil {
			continue
		}
		out = append(out, l.ToResponse())
	}
	return out
}

type MetadataCountEntity struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BookCount int64  `json:"bookCount"`
	CoverURL  string `json:"coverUrl,omitempty"`
}

func (e *MetadataCountEntity) ToResponse() *response.MetadataCountResponse {
	if e == nil {
		return nil
	}
	return &response.MetadataCountResponse{
		ID:        e.ID,
		Name:      e.Name,
		BookCount: e.BookCount,
		CoverURL:  e.CoverURL,
	}
}

func MetadataCountEntitiesToResponse(entities []*MetadataCountEntity) []*response.MetadataCountResponse {
	out := make([]*response.MetadataCountResponse, 0, len(entities))
	for _, m := range entities {
		if m == nil {
			continue
		}
		out = append(out, m.ToResponse())
	}
	return out
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

func (e *AuthorEntity) ToResponse() *response.AuthorResponse {
	if e == nil {
		return nil
	}
	return &response.AuthorResponse{
		ID:        e.ID,
		Name:      e.Name,
		Bio:       e.Bio,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func AuthorEntitiesToResponse(entities []*AuthorEntity) []*response.AuthorResponse {
	out := make([]*response.AuthorResponse, 0, len(entities))
	for _, a := range entities {
		if a == nil {
			continue
		}
		out = append(out, a.ToResponse())
	}
	return out
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

func (e *TagEntity) ToResponse() *response.TagResponse {
	if e == nil {
		return nil
	}
	return &response.TagResponse{
		ID:        e.ID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
	}
}

func TagEntitiesToResponse(entities []*TagEntity) []*response.TagResponse {
	out := make([]*response.TagResponse, 0, len(entities))
	for _, t := range entities {
		if t == nil {
			continue
		}
		out = append(out, t.ToResponse())
	}
	return out
}
