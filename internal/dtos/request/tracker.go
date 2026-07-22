package request

type ConnectTrackerDto struct {
	Provider    string `json:"provider" validate:"required"`
	AccessToken string `json:"access_token" validate:"required"`
}

type MapTrackerDto struct {
	BookID           int64  `json:"book_id" validate:"required"`
	Provider         string `json:"provider" validate:"required"`
	ExternalSeriesID string `json:"external_series_id" validate:"required"`
}

type SyncProgressDto struct {
	BookID   int64  `json:"book_id" validate:"required"`
	Title    string `json:"title" validate:"required"`
	Progress int    `json:"progress" validate:"required,min=1"`
}
