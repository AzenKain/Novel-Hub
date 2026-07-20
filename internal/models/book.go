package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type BookEntity struct {
	ID           string            `json:"id"`
	LibraryID    string            `json:"libraryId"`
	Title        string            `json:"title"`
	AuthorID     *string           `json:"authorId"`
	AuthorName   *string           `json:"authorName,omitempty"`
	Description  *string           `json:"description"`
	CoverURL     *string           `json:"coverUrl"`
	Status       string            `json:"status"`
	MetadataJSON *string           `json:"metadataJson"`
	Files        []*BookFileEntity `json:"files,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type ChapterEntity struct {
	ID           string    `json:"id"`
	BookID       string    `json:"bookId"`
	Title        string    `json:"title"`
	ContentPath  *string   `json:"contentPath"`
	ChapterIndex int64     `json:"chapterIndex"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type BookFileEntity struct {
	ID        string    `json:"id"`
	BookID    string    `json:"bookId"`
	Path      string    `json:"path"`
	Format    string    `json:"format"`
	SizeBytes int64     `json:"sizeBytes"`
	ModTime   time.Time `json:"modTime"`
	Hash      *string   `json:"hash"`
	State     *string   `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type FileRefEntity struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	BookID string `json:"bookId"`
}

type DuplicateFileEntity struct {
	Hash           *string `json:"hash"`
	DuplicateCount int64   `json:"duplicateCount"`
	FileIDs        string  `json:"fileIds"`
}

type FTSResultEntity struct {
	BookID    string `json:"bookId"`
	ChapterID string `json:"chapterId"`
	Title     string `json:"title"`
}

type SeriesEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type PublisherEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type LanguageEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type MetadataCountEntity struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BookCount int64  `json:"bookCount"`
	CoverURL  string `json:"coverUrl,omitempty"`
}

type ReaderBootstrapEntity struct {
	Book     *BookEntity      `json:"book"`
	Chapters []*ChapterEntity `json:"chapters"`
}

type ReaderAssetEntity struct {
	Data        []byte `json:"-"`
	ContentType string `json:"contentType"`
}

func (e *BookEntity) FromSqlc(res sqlc.Book) *BookEntity {
	if e == nil {
		e = &BookEntity{}
	}
	status := ""
	if res.Status.Valid {
		status = res.Status.String
	}
	e.ID = res.ID
	e.LibraryID = res.LibraryID
	e.Title = res.Title
	e.AuthorID = convert.NullStringToStrPtr(res.AuthorID)
	e.Description = convert.NullStringToStrPtr(res.Description)
	e.CoverURL = convert.NullStringToStrPtr(res.CoverUrl)
	e.Status = status
	e.MetadataJSON = convert.NullStringToStrPtr(res.MetadataJson)
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type BookEntities []*BookEntity

func (e *BookEntities) FromSqlc(rows []sqlc.Book) []*BookEntity {
	books := make([]*BookEntity, len(rows))
	for i, res := range rows {
		books[i] = (&BookEntity{}).FromSqlc(res)
	}
	return books
}

func (e *ChapterEntity) FromSqlc(res sqlc.Chapter) *ChapterEntity {
	if e == nil {
		e = &ChapterEntity{}
	}
	e.ID = res.ID
	e.BookID = res.BookID
	e.Title = res.Title
	e.ContentPath = convert.NullStringToStrPtr(res.ContentPath)
	e.ChapterIndex = res.ChapterIndex
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type ChapterEntities []*ChapterEntity

func (e *ChapterEntities) FromSqlc(rows []sqlc.Chapter) []*ChapterEntity {
	chapters := make([]*ChapterEntity, len(rows))
	for i, res := range rows {
		chapters[i] = (&ChapterEntity{}).FromSqlc(res)
	}
	return chapters
}

func (e *BookFileEntity) FromSqlc(res sqlc.BookFile) *BookFileEntity {
	if e == nil {
		e = &BookFileEntity{}
	}
	e.ID = res.ID
	e.BookID = res.BookID
	e.Path = res.Path
	e.Format = res.Format
	e.SizeBytes = res.SizeBytes
	e.ModTime = res.ModTime
	e.Hash = convert.NullStringToStrPtr(res.Hash)
	e.State = convert.NullStringToStrPtr(res.State)
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type BookFileEntities []*BookFileEntity

func (e *BookFileEntities) FromSqlc(rows []sqlc.BookFile) []*BookFileEntity {
	files := make([]*BookFileEntity, len(rows))
	for i, res := range rows {
		files[i] = (&BookFileEntity{}).FromSqlc(res)
	}
	return files
}

type BookFileUploadResult struct {
	Uploaded int               `json:"uploaded"`
	Total    int               `json:"total"`
	Files    []*BookFileEntity `json:"files"`
}

func (e *FileRefEntity) FromSqlc(res sqlc.ListAllFilesRow) *FileRefEntity {
	if e == nil {
		e = &FileRefEntity{}
	}
	e.ID = res.ID
	e.Path = res.Path
	e.BookID = res.BookID
	return e
}

type FileRefEntities []*FileRefEntity

func (e *FileRefEntities) FromSqlc(rows []sqlc.ListAllFilesRow) []*FileRefEntity {
	files := make([]*FileRefEntity, len(rows))
	for i, res := range rows {
		files[i] = (&FileRefEntity{}).FromSqlc(res)
	}
	return files
}

func (e *DuplicateFileEntity) FromSqlc(res sqlc.GetDuplicateFilesRow) *DuplicateFileEntity {
	if e == nil {
		e = &DuplicateFileEntity{}
	}
	e.Hash = convert.NullStringToStrPtr(res.Hash)
	e.DuplicateCount = res.DuplicateCount
	e.FileIDs = res.FileIds
	return e
}

type DuplicateFileEntities []*DuplicateFileEntity

func (e *DuplicateFileEntities) FromSqlc(rows []sqlc.GetDuplicateFilesRow) []*DuplicateFileEntity {
	files := make([]*DuplicateFileEntity, len(rows))
	for i, res := range rows {
		files[i] = (&DuplicateFileEntity{}).FromSqlc(res)
	}
	return files
}

func (e *FTSResultEntity) FromSqlc(res sqlc.SearchFTSRow) *FTSResultEntity {
	if e == nil {
		e = &FTSResultEntity{}
	}
	e.BookID = res.BookID
	e.ChapterID = res.ChapterID
	e.Title = res.Title
	return e
}

type FTSResultEntities []*FTSResultEntity

func (e *FTSResultEntities) FromSqlc(rows []sqlc.SearchFTSRow) []*FTSResultEntity {
	results := make([]*FTSResultEntity, len(rows))
	for i, res := range rows {
		results[i] = (&FTSResultEntity{}).FromSqlc(res)
	}
	return results
}

func (e *SeriesEntity) FromSqlc(res sqlc.Series) *SeriesEntity {
	if e == nil {
		e = &SeriesEntity{}
	}
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	return e
}

func (e *PublisherEntity) FromSqlc(res sqlc.Publisher) *PublisherEntity {
	if e == nil {
		e = &PublisherEntity{}
	}
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	return e
}

func (e *LanguageEntity) FromSqlc(res sqlc.Language) *LanguageEntity {
	if e == nil {
		e = &LanguageEntity{}
	}
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	return e
}

func (e *AuthorEntity) FromSqlc(res sqlc.Author) *AuthorEntity {
	if e == nil {
		e = &AuthorEntity{}
	}
	e.ID = res.ID
	e.Name = res.Name
	e.Bio = convert.NullStringToStrPtr(res.Bio)
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

func (e *TagEntity) FromSqlc(res sqlc.Tag) *TagEntity {
	if e == nil {
		e = &TagEntity{}
	}
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	return e
}

type AuthorEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Bio       *string   `json:"bio"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TagEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
