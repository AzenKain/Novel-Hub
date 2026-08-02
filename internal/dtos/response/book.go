package response

import "time"

type BookFileResponse struct {
	ID        string    `json:"id"`
	BookID    string    `json:"book_id"`
	Path      string    `json:"path"`
	Format    string    `json:"format"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"mod_time"`
	Hash      *string   `json:"hash,omitempty"`
	State     *string   `json:"state,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BookFileUploadResultResponse struct {
	Uploaded int                 `json:"uploaded"`
	Total    int                 `json:"total"`
	Files    []*BookFileResponse `json:"files,omitempty"`
}

type FileRefResponse struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	BookID string `json:"book_id"`
}

type DuplicateFileResponse struct {
	Hash           *string `json:"hash,omitempty"`
	DuplicateCount int64   `json:"duplicate_count"`
	FileIDs        string  `json:"file_ids"`
}

type DuplicateFileDetailResponse struct {
	FileID       string  `json:"file_id"`
	BookID       string  `json:"book_id"`
	BookTitle    string  `json:"book_title"`
	BookCoverURL *string `json:"book_cover_url,omitempty"`
	LibraryID    string  `json:"library_id"`
	Format       string  `json:"format"`
	SizeBytes    int64   `json:"size_bytes"`
	Path         string  `json:"path"`
	CreatedAt    string  `json:"created_at"`
}

type DuplicateGroupResponse struct {
	Hash           string                         `json:"hash"`
	DuplicateCount int                            `json:"duplicate_count"`
	Files          []*DuplicateFileDetailResponse `json:"files"`
}

type ChapterResponse struct {
	ID           string    `json:"id"`
	BookID       string    `json:"book_id"`
	Title        string    `json:"title"`
	ContentPath  *string   `json:"content_path,omitempty"`
	ChapterIndex int64     `json:"chapter_index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BookResponse struct {
	ID           string              `json:"id"`
	LibraryID    string              `json:"library_id"`
	Title        string              `json:"title"`
	AuthorID     *string             `json:"author_id,omitempty"`
	AuthorName   *string             `json:"author_name,omitempty"`
	Description  *string             `json:"description,omitempty"`
	CoverURL     *string             `json:"cover_url,omitempty"`
	Status       string              `json:"status"`
	MetadataJSON *string             `json:"metadata_json,omitempty"`
	Files        []*BookFileResponse `json:"files,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type FTSResultResponse struct {
	BookID    string `json:"book_id"`
	ChapterID string `json:"chapter_id"`
	Title     string `json:"title"`
}

type ReaderBootstrapResponse struct {
	Book     *BookResponse      `json:"book"`
	Chapters []*ChapterResponse `json:"chapters"`
}

type ReaderAssetResponse struct {
	ContentType string `json:"content_type"`
}
