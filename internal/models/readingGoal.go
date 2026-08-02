package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
)

type ReadingGoalEntity struct {
	UserID             string    `json:"userId"`
	TargetWordsPerDay  int64     `json:"targetWordsPerDay"`
	TargetBooksPerYear int64     `json:"targetBooksPerYear"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
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
