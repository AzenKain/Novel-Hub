package response

import "time"

type PodcastResponse struct {
	ID            string     `json:"id"`
	LibraryID     string     `json:"library_id"`
	FeedURL       string     `json:"feed_url"`
	Title         string     `json:"title"`
	Description   *string    `json:"description,omitempty"`
	CoverURL      *string    `json:"cover_url,omitempty"`
	Author        *string    `json:"author,omitempty"`
	AutoDownload  bool       `json:"auto_download"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	EpisodeCount  int64      `json:"episode_count,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type PodcastEpisodeResponse struct {
	ID          string     `json:"id"`
	PodcastID   string     `json:"podcast_id"`
	GUID        string     `json:"guid"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	AudioURL    string     `json:"audio_url"`
	DurationSec *int64     `json:"duration_sec,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Downloaded  bool       `json:"downloaded"`
	BookID      *string    `json:"book_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
