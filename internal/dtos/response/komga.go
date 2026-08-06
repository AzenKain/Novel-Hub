package response

// Field names come from two Kotlin clients that must both decode the same JSON: the Mihon
// extension (keiyoushi/extensions-source .../komga/dto) and Mihon's built-in tracker
// (mihonapp/mihon .../data/track/komga). Their DTOs disagree, so this emits the union — and a
// non-nullable Kotlin field that is absent throws at decode time, hence no omitempty.

type KomgaPageWrapper[T any] struct {
	Content          []T   `json:"content"`
	Empty            bool  `json:"empty"`
	First            bool  `json:"first"`
	Last             bool  `json:"last"`
	Number           int64 `json:"number"`
	NumberOfElements int64 `json:"numberOfElements"`
	Size             int64 `json:"size"`
	TotalElements    int64 `json:"totalElements"`
	TotalPages       int64 `json:"totalPages"`
}

type KomgaLibrary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type KomgaAuthor struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type KomgaSeriesMetadata struct {
	Status               string   `json:"status"`
	Created              *string  `json:"created"`
	LastModified         *string  `json:"lastModified"`
	Title                string   `json:"title"`
	TitleSort            string   `json:"titleSort"`
	Summary              string   `json:"summary"`
	SummaryLock          bool     `json:"summaryLock"`
	ReadingDirection     string   `json:"readingDirection"`
	ReadingDirectionLock bool     `json:"readingDirectionLock"`
	Publisher            string   `json:"publisher"`
	PublisherLock        bool     `json:"publisherLock"`
	AgeRating            *int     `json:"ageRating"`
	AgeRatingLock        bool     `json:"ageRatingLock"`
	Language             string   `json:"language"`
	LanguageLock         bool     `json:"languageLock"`
	Genres               []string `json:"genres"`
	GenresLock           bool     `json:"genresLock"`
	Tags                 []string `json:"tags"`
	TagsLock             bool     `json:"tagsLock"`
	TotalBookCount       *int     `json:"totalBookCount"`
}

type KomgaBookMetadataAggregation struct {
	Authors       []KomgaAuthor `json:"authors"`
	Tags          []string      `json:"tags"`
	ReleaseDate   *string       `json:"releaseDate"`
	Summary       string        `json:"summary"`
	SummaryNumber string        `json:"summaryNumber"`
	Created       string        `json:"created"`
	LastModified  string        `json:"lastModified"`
}

type KomgaSeries struct {
	ID               string                       `json:"id"`
	LibraryID        string                       `json:"libraryId"`
	Name             string                       `json:"name"`
	Created          *string                      `json:"created"`
	LastModified     *string                      `json:"lastModified"`
	FileLastModified string                       `json:"fileLastModified"`
	BooksCount       int                          `json:"booksCount"`
	BooksReadCount   int                          `json:"booksReadCount"`
	BooksUnreadCount int                          `json:"booksUnreadCount"`
	BooksInProgress  int                          `json:"booksInProgressCount"`
	Metadata         KomgaSeriesMetadata          `json:"metadata"`
	BooksMetadata    KomgaBookMetadataAggregation `json:"booksMetadata"`
}

type KomgaMedia struct {
	Status               string `json:"status"`
	MediaType            string `json:"mediaType"`
	PagesCount           int    `json:"pagesCount"`
	MediaProfile         string `json:"mediaProfile"`
	EpubDivinaCompatible bool   `json:"epubDivinaCompatible"`
}

type KomgaBookMetadata struct {
	Title          string        `json:"title"`
	TitleLock      bool          `json:"titleLock"`
	Summary        string        `json:"summary"`
	SummaryLock    bool          `json:"summaryLock"`
	Number         string        `json:"number"`
	NumberLock     bool          `json:"numberLock"`
	NumberSort     float64       `json:"numberSort"`
	NumberSortLock bool          `json:"numberSortLock"`
	ReleaseDate    *string       `json:"releaseDate"`
	ReleaseDateLck bool          `json:"releaseDateLock"`
	Authors        []KomgaAuthor `json:"authors"`
	AuthorsLock    bool          `json:"authorsLock"`
	Tags           []string      `json:"tags"`
	TagsLock       bool          `json:"tagsLock"`
}

type KomgaBook struct {
	ID               string            `json:"id"`
	SeriesID         string            `json:"seriesId"`
	SeriesTitle      string            `json:"seriesTitle"`
	Name             string            `json:"name"`
	Number           float64           `json:"number"`
	Created          *string           `json:"created"`
	LastModified     *string           `json:"lastModified"`
	FileLastModified string            `json:"fileLastModified"`
	SizeBytes        int64             `json:"sizeBytes"`
	Size             string            `json:"size"`
	Media            KomgaMedia        `json:"media"`
	Metadata         KomgaBookMetadata `json:"metadata"`
}

type KomgaPage struct {
	Number    int    `json:"number"`
	FileName  string `json:"fileName"`
	MediaType string `json:"mediaType"`
}

type KomgaReadProgressV2 struct {
	BooksCount                  int     `json:"booksCount"`
	BooksReadCount              int     `json:"booksReadCount"`
	BooksUnreadCount            int     `json:"booksUnreadCount"`
	BooksInProgressCount        int     `json:"booksInProgressCount"`
	LastReadContinuousNumberSrt float64 `json:"lastReadContinuousNumberSort"`
	MaxNumberSort               float64 `json:"maxNumberSort"`
}

type KomgaReadProgressUpdateV2 struct {
	LastBookNumberSortRead float64 `json:"lastBookNumberSortRead"`
}
