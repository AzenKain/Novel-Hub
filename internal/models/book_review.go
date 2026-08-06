package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type BookReviewEntity struct {
	UserID    string     `json:"user_id"`
	BookID    string     `json:"book_id"`
	Rating    int64      `json:"rating"`
	Review    *string    `json:"review,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	UserName  string     `json:"user_name,omitempty"`
	UserEmail string     `json:"user_email,omitempty"`
	BookTitle string     `json:"book_title,omitempty"`
}

func (e *BookReviewEntity) FromSqlc(res sqlc.BookReview) *BookReviewEntity {
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.Rating = res.Rating
	e.Review = convert.NullStringToStrPtr(res.Review)
	e.CreatedAt = &res.CreatedAt
	e.UpdatedAt = &res.UpdatedAt
	return e
}

type BookReviewEntities []*BookReviewEntity

func (e *BookReviewEntities) FromSqlc(rows []sqlc.BookReview) []*BookReviewEntity {
	slice := make([]*BookReviewEntity, len(rows))
	flat := make([]BookReviewEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (e *BookReviewEntity) FromListAllReviewsSqlc(res sqlc.ListAllReviewsRow) *BookReviewEntity {
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.Rating = res.Rating
	e.Review = convert.NullStringToStrPtr(res.Review)
	e.CreatedAt = &res.CreatedAt
	e.UpdatedAt = &res.UpdatedAt
	e.UserName = convert.NullStringToString(res.UserName)
	e.UserEmail = res.UserEmail
	e.BookTitle = res.BookTitle
	return e
}

func (e *BookReviewEntities) FromListAllReviewsSqlc(rows []sqlc.ListAllReviewsRow) []*BookReviewEntity {
	slice := make([]*BookReviewEntity, len(rows))
	flat := make([]BookReviewEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromListAllReviewsSqlc(row)
	}
	return slice
}

func (e *BookReviewEntity) ToResponse() *response.BookReviewResponse {
	if e == nil {
		return nil
	}
	return &response.BookReviewResponse{
		UserID:    e.UserID,
		BookID:    e.BookID,
		Rating:    e.Rating,
		Review:    e.Review,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		UserName:  e.UserName,
		UserEmail: e.UserEmail,
		BookTitle: e.BookTitle,
	}
}

func BookReviewEntitiesToResponse(entities []*BookReviewEntity) []*response.BookReviewResponse {
	out := make([]*response.BookReviewResponse, 0, len(entities))
	for _, r := range entities {
		if r == nil {
			continue
		}
		out = append(out, r.ToResponse())
	}
	return out
}
