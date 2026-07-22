package request

type ImportCalibreDto struct {
	Path      string `json:"path" validate:"required,min=1"`
	LibraryID string `json:"library_id" validate:"omitempty"`
}
