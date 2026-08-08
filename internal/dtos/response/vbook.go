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
