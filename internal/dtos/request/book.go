package request

type SearchBookDto struct {
	PaginationDto
	Search     string `json:"search,omitempty" query:"search" validate:"omitempty"`
	LibraryID  string `json:"library_id,omitempty" query:"library_id" validate:"omitempty"`
	Nav        string `json:"nav,omitempty" query:"nav" validate:"omitempty"`
	Collection string `json:"collection,omitempty" query:"collection" validate:"omitempty"`
	Chip       string `json:"chip,omitempty" query:"chip" validate:"omitempty"`
	Facet      string `json:"facet,omitempty" query:"facet" validate:"omitempty"`
	FacetID    string `json:"facet_id,omitempty" query:"facet_id" validate:"omitempty"`
}

type SearchDeepDto struct {
	PaginationDto
	Query string `json:"q,omitempty" query:"q" validate:"required,min=1,max=200"`
}

type UpdateBookMetadataDto struct {
	Title       string   `json:"title" validate:"required"`
	Author      string   `json:"author" validate:"omitempty"`
	Description string   `json:"description" validate:"omitempty"`
	Publisher   string   `json:"publisher" validate:"omitempty"`
	Language    string   `json:"language" validate:"omitempty"`
	Date        string   `json:"date" validate:"omitempty"`
	Subjects    []string `json:"subjects" validate:"omitempty"`
	Series      string   `json:"series" validate:"omitempty"`
	SeriesIndex string   `json:"series_index" validate:"omitempty"`
}

type ArchiveBookDto struct {
	Archived bool `json:"archived"`
}

type UpdateCoverDto struct {
	UploadedFileName string
	UploadedData     []byte
	CoverURL         string
	EPUBImagePath    string
}

type SearchInBookQueryDto struct {
	Query string `json:"q,omitempty" query:"q" validate:"required,min=1,max=200"`
}

type BookFileQueryDto struct {
	FileID string `json:"file_id,omitempty" query:"file_id" validate:"omitempty"`
}
