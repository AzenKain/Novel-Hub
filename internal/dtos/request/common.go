package request

type PaginationDto struct {
	Page   int    `json:"page,omitempty" query:"page" validate:"omitempty,min=1"`
	Limit  int    `json:"limit,omitempty" query:"limit" validate:"omitempty,min=1,max=100"`
	Offset int    `json:"offset,omitempty" query:"offset" validate:"omitempty,min=0"`
	Order  string `json:"order,omitempty" query:"order" validate:"omitempty,oneof=asc desc"`
	Cursor string `json:"cursor,omitempty" query:"cursor" validate:"omitempty"`
}

type LimitDto struct {
	Limit int `json:"limit,omitempty" query:"limit" validate:"omitempty,min=1,max=100"`
}
