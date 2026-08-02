package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
)

type ReadingGoalEntity struct {
	UserID             string    `json:"user_id"`
	TargetWordsPerDay  int64     `json:"target_words_per_day"`
	TargetBooksPerYear int64     `json:"target_books_per_year"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (e *ReadingGoalEntity) FromSqlc(res sqlc.ReadingGoal) *ReadingGoalEntity {
	e.UserID = res.UserID
	e.TargetWordsPerDay = res.TargetWordsPerDay
	e.TargetBooksPerYear = res.TargetBooksPerYear
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

func (e *ReadingGoalEntity) ToResponse() *response.ReadingGoalResponse {
	if e == nil {
		return nil
	}
	return &response.ReadingGoalResponse{
		UserID:             e.UserID,
		TargetWordsPerDay:  e.TargetWordsPerDay,
		TargetBooksPerYear: e.TargetBooksPerYear,
		UpdatedAt:          e.UpdatedAt,
	}
}
