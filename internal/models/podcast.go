package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type PodcastEntity struct {
	ID            string     `json:"id"`
	LibraryID     string     `json:"library_id"`
	FeedURL       string     `json:"feed_url"`
	Title         string     `json:"title"`
	Description   *string    `json:"description,omitempty"`
	CoverURL      *string    `json:"cover_url,omitempty"`
	Author        *string    `json:"author,omitempty"`
	AutoDownload  bool       `json:"auto_download"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

func (e *PodcastEntity) FromSqlc(r sqlc.Podcast) *PodcastEntity {
	e.ID = r.ID
	e.LibraryID = r.LibraryID
	e.FeedURL = r.FeedUrl
	e.Title = r.Title
	e.Description = convert.NullStringToStrPtr(r.Description)
	e.CoverURL = convert.NullStringToStrPtr(r.CoverUrl)
	e.Author = convert.NullStringToStrPtr(r.Author)
	e.AutoDownload = r.AutoDownload != 0
	e.LastCheckedAt = convert.NullTimeToTimePtr(r.LastCheckedAt)
	e.CreatedAt = convert.NullTimeToTimePtr(r.CreatedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(r.UpdatedAt)
	return e
}

func (e *PodcastEntity) ToResponse() *response.PodcastResponse {
	resp := &response.PodcastResponse{
		ID:           e.ID,
		LibraryID:    e.LibraryID,
		FeedURL:      e.FeedURL,
		Title:        e.Title,
		Description:  e.Description,
		CoverURL:     e.CoverURL,
		Author:       e.Author,
		AutoDownload: e.AutoDownload,
		LastCheckedAt: e.LastCheckedAt,
	}
	if e.CreatedAt != nil {
		resp.CreatedAt = *e.CreatedAt
	}
	if e.UpdatedAt != nil {
		resp.UpdatedAt = *e.UpdatedAt
	}
	return resp
}

type PodcastWithCountEntity struct {
	PodcastEntity
	EpisodeCount int64 `json:"episode_count"`
}

func (e *PodcastWithCountEntity) FromSqlc(r sqlc.ListPodcastsWithCountsRow) *PodcastWithCountEntity {
	e.PodcastEntity = *(&PodcastEntity{}).FromSqlc(sqlc.Podcast{
		ID:            r.ID,
		LibraryID:     r.LibraryID,
		FeedUrl:       r.FeedUrl,
		Title:         r.Title,
		Description:   r.Description,
		CoverUrl:      r.CoverUrl,
		Author:        r.Author,
		AutoDownload:  r.AutoDownload,
		LastCheckedAt: r.LastCheckedAt,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	})
	e.EpisodeCount = r.EpisodeCount
	return e
}

type PodcastEpisodeEntity struct {
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
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

func (e *PodcastEpisodeEntity) FromSqlc(r sqlc.PodcastEpisode) *PodcastEpisodeEntity {
	e.ID = r.ID
	e.PodcastID = r.PodcastID
	e.GUID = r.Guid
	e.Title = r.Title
	e.Description = convert.NullStringToStrPtr(r.Description)
	e.AudioURL = r.AudioUrl
	if r.DurationSec.Valid {
		v := r.DurationSec.Int64
		e.DurationSec = &v
	}
	e.PublishedAt = convert.NullTimeToTimePtr(r.PublishedAt)
	e.Downloaded = r.Downloaded != 0
	e.BookID = convert.NullStringToStrPtr(r.BookID)
	e.CreatedAt = convert.NullTimeToTimePtr(r.CreatedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(r.UpdatedAt)
	return e
}

func (e *PodcastEpisodeEntity) ToResponse() *response.PodcastEpisodeResponse {
	resp := &response.PodcastEpisodeResponse{
		ID:          e.ID,
		PodcastID:   e.PodcastID,
		GUID:        e.GUID,
		Title:       e.Title,
		Description: e.Description,
		AudioURL:    e.AudioURL,
		DurationSec: e.DurationSec,
		PublishedAt: e.PublishedAt,
		Downloaded:  e.Downloaded,
		BookID:      e.BookID,
	}
	if e.CreatedAt != nil {
		resp.CreatedAt = *e.CreatedAt
	}
	if e.UpdatedAt != nil {
		resp.UpdatedAt = *e.UpdatedAt
	}
	return resp
}

type PodcastEpisodeEntities []*PodcastEpisodeEntity

func (e *PodcastEpisodeEntities) FromSqlc(rows []sqlc.PodcastEpisode) []*PodcastEpisodeEntity {
	out := make([]*PodcastEpisodeEntity, len(rows))
	for i, row := range rows {
		out[i] = (&PodcastEpisodeEntity{}).FromSqlc(row)
	}
	return out
}