package services

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/convert"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
	"novelhub/pkg/worker"
)

const (
	podcastRefreshJobType  = "podcast_refresh"
	podcastDownloadJobType = "podcast_download"
	maxFeedBytes           = 250 << 20
)

type PodcastService interface {
	ListPodcasts(ctx context.Context) ([]*response.PodcastResponse, error)
	Subscribe(ctx context.Context, feedURL, libraryID string) (*response.PodcastResponse, error)
	UpdatePodcast(ctx context.Context, id string, autoDownload *bool) (*response.PodcastResponse, error)
	DeletePodcast(ctx context.Context, id string) error
	ListEpisodes(ctx context.Context, podcastID string) ([]*response.PodcastEpisodeResponse, error)
	QueueRefreshPodcast(ctx context.Context, podcastID string) (string, error)
	RefreshPodcast(ctx context.Context, podcastID string) error
	RefreshAllPodcasts(ctx context.Context) error
	DownloadEpisode(ctx context.Context, podcastID, episodeID string) (string, error)
	ExecutePodcastRefreshJob(ctx context.Context, payloadJSON string) error
	ExecutePodcastDownloadJob(ctx context.Context, payloadJSON string) error
}

type podcastRefreshPayload struct {
	PodcastID string `json:"podcast_id"`
}

type podcastDownloadPayload struct {
	PodcastID string `json:"podcast_id"`
	EpisodeID string `json:"episode_id"`
}

type podcastService struct {
	repo           repositories.PodcastRepository
	bookRepo       repositories.BookDBRepository
	fileRepo       repositories.BookFileRepository
	libraryRepo    repositories.LibraryRepository
	jobQueue       *worker.Queue
	httpClient     *http.Client
	downloadClient *http.Client
	txManager      database.TxManager
}

func NewPodcastService(repo repositories.PodcastRepository, bookRepo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, libraryRepo repositories.LibraryRepository, jobQueue *worker.Queue, txManager ...database.TxManager) PodcastService {
	var txMgr database.TxManager
	if len(txManager) > 0 {
		txMgr = txManager[0]
	}
	return &podcastService{
		repo:           repo,
		bookRepo:       bookRepo,
		fileRepo:       fileRepo,
		libraryRepo:    libraryRepo,
		jobQueue:       jobQueue,
		httpClient:     netx.NewSafeHTTPClient(30 * time.Second),
		downloadClient: netx.NewSafeHTTPClient(5 * time.Minute),
		txManager:      txMgr,
	}
}

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title        string    `xml:"title"`
	Description  string    `xml:"description"`
	Author       string    `xml:"author"`
	ItunesAuthor string    `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author"`
	ItunesImage  rssImage  `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Items        []rssItem `xml:"item"`
}

type rssImage struct {
	Href string `xml:"href,attr"`
}

type rssFeedLegacy struct {
	Channel struct {
		ImageURL string `xml:"image>url"`
	} `xml:"channel"`
}

type rssItem struct {
	Title          string       `xml:"title"`
	GUID           string       `xml:"guid"`
	Description    string       `xml:"description"`
	PubDate        string       `xml:"pubDate"`
	Enclosure      rssEnclosure `xml:"enclosure"`
	ItunesTitle    string       `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd title"`
	ItunesDuration string       `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type parsedPodcast struct {
	Title       string
	Description string
	CoverURL    string
	Author      string
	Episodes    []parsedEpisode
}

type parsedEpisode struct {
	GUID        string
	Title       string
	Description string
	AudioURL    string
	DurationSec *int64
	PublishedAt *time.Time
}

func parseFeed(data []byte) (*parsedPodcast, error) {
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "invalid RSS feed")
	}
	ch := feed.Channel
	if ch.Title == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "feed has no channel title")
	}

	if isTemplate(ch.Title) || isTemplate(ch.Author) || isTemplate(ch.ItunesAuthor) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "this RSS feed is an uncompiled template (e.g. Jekyll source). Please subscribe to the compiled feed URL instead.")
	}

	author := ch.Author
	if author == "" {
		author = ch.ItunesAuthor
	}
	cover := ch.ItunesImage.Href
	if cover == "" {
		var legacy rssFeedLegacy
		if err := xml.Unmarshal(data, &legacy); err == nil {
			cover = legacy.Channel.ImageURL
		}
	}

	if isTemplate(cover) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "this RSS feed is an uncompiled template (e.g. Jekyll source). Please subscribe to the compiled feed URL instead.")
	}

	out := &parsedPodcast{Title: ch.Title, Description: ch.Description, CoverURL: cover, Author: author}
	for _, item := range ch.Items {
		title := item.ItunesTitle
		if title == "" {
			title = item.Title
		}
		if item.GUID == "" || item.Enclosure.URL == "" {
			continue
		}
		if isTemplate(title) || isTemplate(item.Enclosure.URL) {
			return nil, apperrors.New(apperrors.ErrBadRequest, "this RSS feed is an uncompiled template (e.g. Jekyll source). Please subscribe to the compiled feed URL instead.")
		}

		out.Episodes = append(out.Episodes, parsedEpisode{
			GUID:        item.GUID,
			Title:       title,
			Description: item.Description,
			AudioURL:    item.Enclosure.URL,
			DurationSec: parseItunesDuration(item.ItunesDuration),
			PublishedAt: parseFeedTime(item.PubDate),
		})
	}
	return out, nil
}

func isTemplate(s string) bool {
	return strings.Contains(s, "{{") || strings.Contains(s, "{%")
}

func parseItunesDuration(v string) *int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	var total int64
	for _, part := range strings.Split(v, ":") {
		n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil
		}
		total = total*60 + n
	}
	return &total
}

var feedTimeLayouts = []string{time.RFC1123Z, time.RFC1123, time.RFC3339, "2006-01-02T15:04:05-07:00", "Mon, 02 Jan 2006 15:04:05 MST", "2006-01-02"}

func parseFeedTime(v string) *time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	for _, layout := range feedTimeLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return &t
		}
	}
	return nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *podcastService) fetchFeed(ctx context.Context, feedURL string) (*parsedPodcast, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "invalid feed URL")
	}
	req.Header.Set("User-Agent", "NovelHub/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "failed to fetch feed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "failed to read feed")
	}
	if resp.StatusCode >= 400 {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("feed returned HTTP %d", resp.StatusCode))
	}
	return parseFeed(body)
}

func (s *podcastService) ListPodcasts(ctx context.Context) ([]*response.PodcastResponse, error) {
	rows, err := s.repo.ListPodcastsWithCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.PodcastResponse, len(rows))
	for i, row := range rows {
		resp := row.PodcastEntity.ToResponse()
		resp.EpisodeCount = row.EpisodeCount
		out[i] = resp
	}
	return out, nil
}

func (s *podcastService) Subscribe(ctx context.Context, feedURL, libraryID string) (*response.PodcastResponse, error) {
	if _, err := s.libraryRepo.GetLibrary(ctx, libraryID); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "library not found")
	}
	existing, err := s.repo.GetPodcastByFeedURL(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "feed already subscribed")
	}

	feed, err := s.fetchFeed(ctx, feedURL)
	if err != nil {
		return nil, err
	}

	var txRepo repositories.PodcastRepository = s.repo
	var tx *sql.Tx
	if s.txManager != nil {
		var err error
		tx, err = s.txManager.BeginTx(ctx, nil)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "failed to begin transaction")
		}
		defer func() {
			if tx != nil {
				_ = tx.Rollback()
			}
		}()
		txRepo = s.repo.WithTx(tx)
	}

	id := uuid.Must(uuid.NewV7()).String()
	podcast, err := txRepo.CreatePodcast(ctx, id, libraryID, feedURL, feed.Title,
		strPtrOrNil(feed.Description), strPtrOrNil(feed.CoverURL), strPtrOrNil(feed.Author))
	if err != nil {
		return nil, err
	}
	for _, ep := range feed.Episodes {
		if _, err := txRepo.UpsertEpisode(ctx, uuid.Must(uuid.NewV7()).String(), id, ep.GUID, ep.Title,
			strPtrOrNil(ep.Description), ep.AudioURL, ep.DurationSec, ep.PublishedAt); err != nil {
			return nil, err
		}
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "failed to commit transaction")
		}
		tx = nil
	}
	return podcast.ToResponse(), nil
}

func (s *podcastService) UpdatePodcast(ctx context.Context, id string, autoDownload *bool) (*response.PodcastResponse, error) {
	podcast, err := s.repo.GetPodcast(ctx, id)
	if err != nil {
		return nil, err
	}
	if podcast == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "podcast not found")
	}
	if autoDownload != nil {
		podcast.AutoDownload = *autoDownload
	}
	updated, err := s.repo.UpdatePodcast(ctx, id, podcast.Title, podcast.Description, podcast.CoverURL, podcast.Author, podcast.AutoDownload, podcast.LastCheckedAt)
	if err != nil {
		return nil, err
	}
	return updated.ToResponse(), nil
}

func (s *podcastService) DeletePodcast(ctx context.Context, id string) error {
	return s.repo.DeletePodcast(ctx, id)
}

func (s *podcastService) ListEpisodes(ctx context.Context, podcastID string) ([]*response.PodcastEpisodeResponse, error) {
	episodes, err := s.repo.ListEpisodes(ctx, podcastID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.PodcastEpisodeResponse, len(episodes))
	for i, ep := range episodes {
		out[i] = ep.ToResponse()
	}
	return out, nil
}

func (s *podcastService) QueueRefreshPodcast(ctx context.Context, podcastID string) (string, error) {
	payload, err := jsonx.MarshalString(podcastRefreshPayload{PodcastID: podcastID})
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to marshal refresh payload")
	}
	jobID := uuid.Must(uuid.NewV7()).String()
	if s.jobQueue != nil {
		if err := s.jobQueue.Enqueue(ctx, worker.Job{ID: jobID, Type: podcastRefreshJobType, Payload: payload}); err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to enqueue refresh job")
		}
		return jobID, nil
	}
	return jobID, s.ExecutePodcastRefreshJob(ctx, payload)
}

func (s *podcastService) RefreshPodcast(ctx context.Context, podcastID string) error {
	podcast, err := s.repo.GetPodcast(ctx, podcastID)
	if err != nil {
		return err
	}
	if podcast == nil {
		return apperrors.New(apperrors.ErrNotFound, "podcast not found")
	}

	feed, err := s.fetchFeed(ctx, podcast.FeedURL)
	if err != nil {
		return err
	}

	existingGuids, err := s.repo.ListEpisodeGuids(ctx, podcastID)
	if err != nil {
		return err
	}
	known := make(map[string]struct{}, len(existingGuids))
	for _, g := range existingGuids {
		known[g] = struct{}{}
	}

	now := time.Now()
	if _, err := s.repo.UpdatePodcast(ctx, podcastID, podcast.Title, podcast.Description, podcast.CoverURL, podcast.Author, podcast.AutoDownload, &now); err != nil {
		return err
	}

	for _, ep := range feed.Episodes {
		created, err := s.repo.UpsertEpisode(ctx, uuid.Must(uuid.NewV7()).String(), podcastID, ep.GUID, ep.Title,
			strPtrOrNil(ep.Description), ep.AudioURL, ep.DurationSec, ep.PublishedAt)
		if err != nil {
			return err
		}
		if podcast.AutoDownload {
			if _, isNew := known[ep.GUID]; !isNew {
				if err := s.enqueueDownload(ctx, podcastID, created.ID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *podcastService) RefreshAllPodcasts(ctx context.Context) error {
	ids, err := s.repo.ListPodcastIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.RefreshPodcast(ctx, id); err != nil {
			log.Error().Err(err).Str("podcast_id", id).Msg("podcast refresh failed; continuing")
		}
	}
	return nil
}

func (s *podcastService) enqueueDownload(ctx context.Context, podcastID, episodeID string) error {
	payload, err := jsonx.MarshalString(podcastDownloadPayload{PodcastID: podcastID, EpisodeID: episodeID})
	if err != nil {
		return err
	}
	jobID := uuid.Must(uuid.NewV7()).String()
	if s.jobQueue != nil {
		return s.jobQueue.Enqueue(ctx, worker.Job{ID: jobID, Type: podcastDownloadJobType, Payload: payload})
	}
	return s.ExecutePodcastDownloadJob(ctx, payload)
}

func (s *podcastService) DownloadEpisode(ctx context.Context, podcastID, episodeID string) (string, error) {
	payload, err := jsonx.MarshalString(podcastDownloadPayload{PodcastID: podcastID, EpisodeID: episodeID})
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to marshal download payload")
	}
	jobID := uuid.Must(uuid.NewV7()).String()
	if s.jobQueue != nil {
		if err := s.jobQueue.Enqueue(ctx, worker.Job{ID: jobID, Type: podcastDownloadJobType, Payload: payload}); err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to enqueue download job")
		}
		return jobID, nil
	}
	return jobID, s.ExecutePodcastDownloadJob(ctx, payload)
}

func (s *podcastService) ExecutePodcastRefreshJob(ctx context.Context, payloadJSON string) error {
	var payload podcastRefreshPayload
	if err := jsonx.UnmarshalString(payloadJSON, &payload); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid refresh payload")
	}
	if payload.PodcastID == "" {
		return s.RefreshAllPodcasts(ctx)
	}
	return s.RefreshPodcast(ctx, payload.PodcastID)
}

func (s *podcastService) ExecutePodcastDownloadJob(ctx context.Context, payloadJSON string) error {
	var payload podcastDownloadPayload
	if err := jsonx.UnmarshalString(payloadJSON, &payload); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid download payload")
	}

	episode, err := s.repo.GetEpisode(ctx, payload.EpisodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.New(apperrors.ErrNotFound, "episode not found")
		}
		return err
	}
	if episode.Downloaded {
		return nil
	}
	podcast, err := s.repo.GetPodcast(ctx, episode.PodcastID)
	if err != nil {
		return err
	}
	if podcast == nil {
		return apperrors.New(apperrors.ErrNotFound, "podcast not found")
	}

	bookID := uuid.Must(uuid.NewV7()).String()
	tmp, err := os.CreateTemp("", "novelhub-podcast-*")
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to create temp file")
	}
	tmpPath := tmp.Name()
	if err := s.downloadAudio(ctx, episode.AudioURL, tmp); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	src, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	saved, err := s.fileRepo.SaveBook(ctx, bookID, episode.Title+audioExtFromURL(episode.AudioURL), src)
	_ = src.Close()
	_ = os.Remove(tmpPath)
	if err != nil {
		return err
	}

	metaData := map[string]string{
		"source":         "podcast",
		"podcast_title":  podcast.Title,
		"guid":           episode.GUID,
		"original_title": episode.Title,
		"uuid":           bookID,
	}
	_ = s.fileRepo.WriteBookMeta(ctx, bookID, metaData)

	var authorID *string
	if podcast.Author != nil {
		if id, err := ensureAuthor(ctx, s.bookRepo, *podcast.Author); err == nil && id != "" {
			authorID = &id
		}
	}

	fileID := uuid.Must(uuid.NewV7()).String()
	state := "managed"
	book := &models.BookEntity{
		ID:          bookID,
		LibraryID:   podcast.LibraryID,
		Title:       episode.Title,
		AuthorID:    authorID,
		Description: episode.Description,
		Status:      "processing",
	}
	err = s.bookRepo.CreateBookWithFile(ctx, book, &sqlc.CreateBookFileParams{
		ID:        fileID,
		BookID:    bookID,
		Path:      saved.Path,
		Format:    saved.Format,
		SizeBytes: saved.SizeBytes,
		ModTime:   saved.ModTime,
		State:     convert.StrPtrToNullString(&state),
	})
	if err != nil {
		_ = s.fileRepo.RemoveBookDir(ctx, bookID)
		return err
	}
	if err := s.repo.MarkEpisodeDownloaded(ctx, episode.ID, bookID); err != nil {
		return err
	}

	if s.jobQueue != nil {
		_ = s.jobQueue.Enqueue(ctx, worker.Job{ID: uuid.Must(uuid.NewV7()).String(), Type: "extract_metadata", Payload: bookID})
		_ = s.jobQueue.Enqueue(ctx, worker.Job{ID: uuid.Must(uuid.NewV7()).String(), Type: "hash_file", Payload: fileID})
	}
	return nil
}

func (s *podcastService) downloadAudio(ctx context.Context, audioURL string, out io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "invalid audio URL")
	}
	req.Header.Set("User-Agent", "NovelHub/1.0")

	resp, err := s.downloadClient.Do(req)
	if err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "failed to download audio")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("audio download returned HTTP %d", resp.StatusCode))
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "failed to save audio")
	}
	return nil
}

func audioExtFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(u.Path))
	switch ext {
	case ".mp3", ".m4a", ".m4b", ".mp4", ".flac", ".ogg", ".opus", ".wav", ".aac", ".webm":
		return ext
	default:
		return ".mp3"
	}
}
