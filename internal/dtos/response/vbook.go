package response

type VBookHomeItem struct {
	Title  string `json:"title"`
	Input  string `json:"input"`
	Script string `json:"script"`
}

type VBookGenreItem struct {
	Title  string `json:"title"`
	Input  string `json:"input"`
	Script string `json:"script"`
}

type VBookBookItem struct {
	Name        string `json:"name"`
	Link        string `json:"link"`
	Cover       string `json:"cover"`
	Description string `json:"description"`
	Host        string `json:"host"`
}

type VBookBookListResponse struct {
	List []*VBookBookItem `json:"list"`
	Next *string          `json:"next,omitempty"`
}

type VBookBookDetailResponse struct {
	Name        string `json:"name"`
	Cover       string `json:"cover"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Detail      string `json:"detail"`
	Host        string `json:"host"`
	Ongoing     bool   `json:"ongoing"`
}

type VBookTOCItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Host string `json:"host"`
}

type VBookChapterContentResponse struct {
	Content string `json:"content"`
}

type VBookPluginMetadata struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Version     int    `json:"version"`
	Source      string `json:"source"`
	Regexp      string `json:"regexp"`
	Description string `json:"description"`
	Locale      string `json:"locale"`
	Language    string `json:"language"`
	Type        string `json:"type"`
}

type VBookPluginScript struct {
	Home   string `json:"home"`
	Genre  string `json:"genre"`
	Detail string `json:"detail"`
	Search string `json:"search"`
	Toc    string `json:"toc"`
	Chap   string `json:"chap"`
}

type VBookPluginResponse struct {
	Metadata VBookPluginMetadata `json:"metadata"`
	Script   VBookPluginScript   `json:"script"`
}

type VBookEntryResponse struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Path        string `json:"path"`
	Version     int    `json:"version"`
	Source      string `json:"source"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Locale      string `json:"locale"`
}

type VBookRegistryMetadata struct {
	Author      string `json:"author"`
	Description string `json:"description"`
}

type VBookRegistryResponse struct {
	Metadata VBookRegistryMetadata `json:"metadata"`
	Data     []*VBookEntryResponse `json:"data"`
}
