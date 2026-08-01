package request

type ConnectTrackerDto struct {
	Provider    string `json:"provider" validate:"required"`
	AccessToken string `json:"access_token" validate:"required"`
}

type MapTrackerDto struct {
	BookID           string `json:"book_id" validate:"required,uuid"`
	Provider         string `json:"provider" validate:"required"`
	ExternalSeriesID string `json:"external_series_id" validate:"required"`
}

type SyncProgressDto struct {
	BookID   string `json:"book_id" validate:"required,uuid"`
	Title    string `json:"title" validate:"required"`
	Progress int    `json:"progress" validate:"omitempty,min=1"`
}
