package response

type BookSeriesResponse struct {
	SeriesID    string  `json:"series_id"`
	SeriesName  string  `json:"series_name"`
	SeriesIndex *string `json:"series_index,omitempty"`
}

type NextInSeriesResponse struct {
	SeriesID    string  `json:"series_id"`
	SeriesName  string  `json:"series_name"`
	BookID      string  `json:"book_id"`
	Title       string  `json:"title"`
	CoverURL    *string `json:"cover_url,omitempty"`
	SeriesIndex *string `json:"series_index,omitempty"`
}

type BookSeriesContextResponse struct {
	Series []*BookSeriesResponse `json:"series"`
	Next   *NextInSeriesResponse `json:"next,omitempty"`
}
