package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type BookReviewEntity struct {
	UserID    int64      `json:"userId"`
	BookID    string     `json:"bookId"`
	Rating    int64      `json:"rating"`
	Review    *string    `json:"review,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UserName  string     `json:"userName,omitempty"`
	UserEmail string     `json:"userEmail,omitempty"`
	BookTitle string     `json:"bookTitle,omitempty"`
}

func (e *BookReviewEntity) FromSqlc(res sqlc.BookReview) *BookReviewEntity {
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.Rating = res.Rating
	e.Review = convert.NullStringToStrPtr(res.Review)
	e.CreatedAt = convert.NullTimeToTimePtr(res.CreatedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
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
	e.CreatedAt = convert.NullTimeToTimePtr(res.CreatedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
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
