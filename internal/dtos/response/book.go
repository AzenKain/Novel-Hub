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

type PotentialDuplicateResponse struct {
	SourceID    string  `json:"source_id"`
	SourceTitle string  `json:"source_title"`
	TargetID    string  `json:"target_id"`
	TargetTitle string  `json:"target_title"`
	AuthorName  string  `json:"author_name"`
	Similarity  float64 `json:"similarity"`
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
	ID              string              `json:"id"`
	LibraryID       string              `json:"library_id"`
	Title           string              `json:"title"`
	AuthorID        *string             `json:"author_id,omitempty"`
	AuthorName      *string             `json:"author_name,omitempty"`
	Description     *string             `json:"description,omitempty"`
	CoverURL        *string             `json:"cover_url,omitempty"`
	Status          string              `json:"status"`
	AgeRating       string              `json:"age_rating"`
	ContentWarnings []string            `json:"content_warnings,omitempty"`
	MetadataJSON    *string             `json:"metadata_json,omitempty"`
	GoogleBooksID   *string             `json:"google_books_id,omitempty"`
	AnilistID       *string             `json:"anilist_id,omitempty"`
	OpenLibraryID   *string             `json:"openlibrary_id,omitempty"`
	Files           []*BookFileResponse `json:"files,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
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

type BookmarkedBooksPageResponse struct {
	Books      []*BookResponse `json:"books"`
	NextCursor string          `json:"next_cursor"`
}

type BookSearchSnippetResponse struct {
	ChapterID    string `json:"chapter_id"`
	ChapterTitle string `json:"chapter_title"`
	ChapterIndex int64  `json:"chapter_index"`
	Snippet      string `json:"snippet"`
	Offset       int    `json:"offset"`
}

type CoverUpdateResponse struct {
	CoverURL string `json:"cover_url"`
}
