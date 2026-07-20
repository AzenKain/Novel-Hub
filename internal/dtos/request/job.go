package request

type ScanLibraryDto struct {
	LibraryID string `json:"library_id" validate:"required"`
}
