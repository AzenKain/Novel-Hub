package response

import (
	"github.com/golang-jwt/jwt/v5"

	"novelhub/pkg/constants"
)

type CommonResponse struct {
	Status  bool   `json:"status"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
	Message string `json:"message,omitempty"`
}

type JWTClaims struct {
	UId          string               `json:"uid"`
	Roles        []constants.RoleType `json:"roles"`
	RoleIDs      []int64              `json:"role_ids"`
	TokenVersion int32                `json:"token_version"`
	TokenType    string               `json:"token_type"`
	jwt.RegisteredClaims
}

type PaginationMeta struct {
	CurrentPage  int    `json:"current_page"`
	PageSize     int    `json:"page_size"`
	TotalRecords int64  `json:"total_records"`
	TotalPages   int    `json:"total_pages"`
	NextCursor   string `json:"next_cursor,omitempty"`
}

type PaginatedResponse struct {
	Status     bool            `json:"status"`
	Message    string          `json:"message,omitempty"`
	Data       any             `json:"data,omitempty"`
	Errors     any             `json:"errors,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

func BuildPaginatedResponse(data any, totalRecords int64, page int, limit int) *PaginatedResponse {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	totalPages := int((totalRecords + int64(limit) - 1) / int64(limit))
	return &PaginatedResponse{
		Status:  true,
		Message: "Success",
		Data:    data,
		Pagination: &PaginationMeta{
			CurrentPage:  page,
			PageSize:     limit,
			TotalRecords: totalRecords,
			TotalPages:   totalPages,
		},
	}
}

func BuildCursorPaginatedResponse(data any, totalRecords int64, limit int, nextCursor string) *PaginatedResponse {
	if limit < 1 {
		limit = 10
	}
	totalPages := 0
	if totalRecords > 0 {
		totalPages = int((totalRecords + int64(limit) - 1) / int64(limit))
	}

	return &PaginatedResponse{
		Status:  true,
		Message: "Success",
		Data:    data,
		Pagination: &PaginationMeta{
			PageSize:     limit,
			TotalRecords: totalRecords,
			TotalPages:   totalPages,
			NextCursor:   nextCursor,
		},
	}
}
