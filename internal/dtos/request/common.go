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

type MetadataFacetDto struct {
	Limit  int    `json:"limit,omitempty" query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string `json:"cursor,omitempty" query:"cursor" validate:"omitempty"`
	Search string `json:"search,omitempty" query:"search" validate:"omitempty,max=200"`
	Alpha  string `json:"alpha,omitempty" query:"alpha" validate:"omitempty,max=8"`
}

// No validate tags: OPDS parses its query string by hand rather than through pkg/validator,
// because a reader device gets an XML error document, not a JSON field-error list.
type OPDSPageDto struct {
	Cursor string
	Limit  int64
}
