package request

type SubscribePodcastDto struct {
	FeedURL   string `json:"feed_url" validate:"required,url"`
	LibraryID string `json:"library_id" validate:"required"`
}

type UpdatePodcastDto struct {
	AutoDownload *bool `json:"auto_download"`
}