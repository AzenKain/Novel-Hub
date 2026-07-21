package response

import "time"

type BookFileResponse struct {
	ID        string    `json:"id"`
	BookID    string    `json:"bookId"`
	Path      string    `json:"path"`
	Format    string    `json:"format"`
	SizeBytes int64     `json:"sizeBytes"`
	ModTime   time.Time `json:"modTime"`
	Hash      *string   `json:"hash,omitempty"`
	State     *string   `json:"state,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type BookFileUploadResultResponse struct {
	Uploaded int                 `json:"uploaded"`
	Total    int                 `json:"total"`
	Files    []*BookFileResponse `json:"files,omitempty"`
}

type FileRefResponse struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	BookID string `json:"bookId"`
}

type DuplicateFileResponse struct {
	Hash           *string `json:"hash,omitempty"`
	DuplicateCount int64   `json:"duplicateCount"`
	FileIDs        string  `json:"fileIds"`
}

type ChapterResponse struct {
	ID           string    `json:"id"`
	BookID       string    `json:"bookId"`
	Title        string    `json:"title"`
	ContentPath  *string   `json:"contentPath,omitempty"`
	ChapterIndex int64     `json:"chapterIndex"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type BookResponse struct {
	ID           string              `json:"id"`
	LibraryID    string              `json:"libraryId"`
	Title        string              `json:"title"`
	AuthorID     *string             `json:"authorId,omitempty"`
	AuthorName   *string             `json:"authorName,omitempty"`
	Description  *string             `json:"description,omitempty"`
	CoverURL     *string             `json:"coverUrl,omitempty"`
	Status       string              `json:"status"`
	MetadataJSON *string             `json:"metadataJson,omitempty"`
	Files        []*BookFileResponse `json:"files,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
}

type FTSResultResponse struct {
	BookID    string `json:"bookId"`
	ChapterID string `json:"chapterId"`
	Title     string `json:"title"`
}

type ReaderBootstrapResponse struct {
	Book     *BookResponse      `json:"book"`
	Chapters []*ChapterResponse `json:"chapters"`
}

type ReaderAssetResponse struct {
	ContentType string `json:"contentType"`
}
