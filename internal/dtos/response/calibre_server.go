package response

type CalibreLibraryInfoResponse struct {
	LibraryMap     map[string]string `json:"library_map"`
	DefaultLibrary string            `json:"default_library"`
}

type CalibreCategorySummary struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	IsCategory bool   `json:"is_category"`
	Count      int64  `json:"count"`
}

type CalibreCategoryItem struct {
	Name        string `json:"name"`
	Count       int64  `json:"count"`
	URL         string `json:"url"`
	HasChildren bool   `json:"has_children"`
}

type CalibreCategoryDetailResponse struct {
	CategoryName  string                `json:"category_name"`
	BaseURL       string                `json:"base_url"`
	TotalNum      int64                 `json:"total_num"`
	Offset        int64                 `json:"offset"`
	Num           int64                 `json:"num"`
	Sort          string                `json:"sort"`
	SortOrder     string                `json:"sort_order"`
	Subcategories []any                 `json:"subcategories"`
	Items         []CalibreCategoryItem `json:"items"`
}

type CalibreBooksInResponse struct {
	TotalNum  int64    `json:"total_num"`
	SortOrder string   `json:"sort_order"`
	Offset    int64    `json:"offset"`
	Num       int64    `json:"num"`
	Sort      string   `json:"sort"`
	BookIDs   []string `json:"book_ids"`
}

type CalibreSearchResponse struct {
	TotalNum              int64    `json:"total_num"`
	SortOrder             string   `json:"sort_order"`
	NumBooksWithoutSearch int64    `json:"num_books_without_search"`
	Offset                int64    `json:"offset"`
	Num                   int64    `json:"num"`
	Sort                  string   `json:"sort"`
	BaseURL               string   `json:"base_url"`
	Query                 string   `json:"query"`
	LibraryID             string   `json:"library_id"`
	BookIDs               []string `json:"book_ids"`
}

type CalibreBookMetadataResponse struct {
	Title        string            `json:"title"`
	Authors      []string          `json:"authors"`
	AuthorSort   string            `json:"author_sort"`
	Series       *string           `json:"series"`
	SeriesIndex  float64           `json:"series_index"`
	Rating       float64           `json:"rating"`
	Tags         []string          `json:"tags"`
	Comments     string            `json:"comments"`
	Publisher    *string           `json:"publisher"`
	Pubdate      string            `json:"pubdate,omitempty"`
	Timestamp    string            `json:"timestamp,omitempty"`
	LastModified string            `json:"last_modified,omitempty"`
	Identifiers  map[string]string `json:"identifiers"`
	Languages    []string          `json:"languages"`
	Cover        string            `json:"cover"`
	Thumbnail    string            `json:"thumbnail"`
	Formats      []string          `json:"formats"`
	MainFormat   map[string]string `json:"main_format,omitempty"`
	OtherFormats map[string]string `json:"other_formats,omitempty"`
	CategoryURLs map[string]any    `json:"category_urls,omitempty"`
}
