package request

type InitUploadDto struct {
	Target      string `json:"target" validate:"required,oneof=library book"`
	LibraryID   string `json:"library_id" validate:"omitempty"`
	BookID      string `json:"book_id" validate:"omitempty"`
	Filename    string `json:"filename" validate:"required,max=255"`
	TotalBytes  int64  `json:"total_bytes" validate:"required,min=1,max=1073741824"`
	TotalChunks int    `json:"total_chunks" validate:"required,min=1,max=100"`
}

type CommitUploadDto struct {
	Target      string `json:"target" validate:"omitempty,oneof=library book"`
	LibraryID   string `json:"library_id" validate:"omitempty"`
	BookID      string `json:"book_id" validate:"omitempty"`
	Filename    string `json:"filename" validate:"omitempty,max=255"`
	TotalChunks int    `json:"total_chunks" validate:"omitempty,min=1,max=100"`
}
