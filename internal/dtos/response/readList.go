package response

import "time"

type ReadListResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	BookCount   int64     `json:"book_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReadListBookResponse struct {
	Position int64         `json:"position"`
	Book     *BookResponse `json:"book"`
}

type ReadListNextResponse struct {
	Position int64         `json:"position"`
	Book     *BookResponse `json:"book"`
	HasNext  bool          `json:"has_next"`
}

type CBLUnmatchedEntry struct {
	Series string `json:"series"`
	Number string `json:"number"`
	Reason string `json:"reason"`
}

type ImportCBLResponse struct {
	ReadList  *ReadListResponse   `json:"read_list"`
	Total     int                 `json:"total"`
	Matched   int                 `json:"matched"`
	Unmatched []CBLUnmatchedEntry `json:"unmatched"`
}
