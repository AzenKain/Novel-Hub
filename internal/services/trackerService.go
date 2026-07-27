package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
)

type TrackerService interface {
	SyncAniListProgress(ctx context.Context, userID string, mediaID string, progress int) error
	SyncMyAnimeListProgress(ctx context.Context, userID string, mangaID string, chaptersRead int) error
	SearchAniListMediaID(ctx context.Context, title string) (string, error)
	GetOrMapBookTrackerID(ctx context.Context, bookID string, title string, provider string) (string, error)
	SaveUserTracker(ctx context.Context, userID string, provider string, accessToken string) error
	SaveBookMapping(ctx context.Context, bookID string, provider string, externalSeriesID string) error
}

type trackerService struct {
	repo       repositories.TrackerRepository
	httpClient *http.Client
}

func NewTrackerService(repo repositories.TrackerRepository) TrackerService {
	return &trackerService{
		repo:       repo,
		httpClient: netx.NewSafeHTTPClient(10 * time.Second),
	}
}

func (s *trackerService) SearchAniListMediaID(ctx context.Context, title string) (string, error) {
	cleanTitle := strings.TrimSpace(title)
	if len(cleanTitle) > 100 {
		cleanTitle = cleanTitle[:100]
	}
	if cleanTitle == "" {
		return "", fmt.Errorf("title cannot be empty")
	}

	if id, err := strconv.ParseInt(cleanTitle, 10, 64); err == nil && id > 0 {
		mediaID, err := s.fetchAniListMediaByID(ctx, id)
		if err == nil && mediaID != "" {
			return mediaID, nil
		}
	}

	return s.fetchAniListMediaBySearch(ctx, cleanTitle)
}

func (s *trackerService) fetchAniListMediaByID(ctx context.Context, id int64) (string, error) {
	query := `query ($id: Int) {
		Media (id: $id, type: MANGA) {
			id
		}
	}`

	payload := map[string]any{
		"query": query,
		"variables": map[string]any{
			"id": id,
		},
	}

	bodyBytes, err := jsonx.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://graphql.anilist.co", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("AniList ID lookup failed with status %d", resp.StatusCode)
	}

	var res struct {
		Data struct {
			Media struct {
				ID int64 `json:"id"`
			} `json:"Media"`
		} `json:"data"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := jsonx.Unmarshal(respBody, &res); err != nil {
		return "", err
	}

	if res.Data.Media.ID == 0 {
		return "", fmt.Errorf("no AniList manga entry found for ID %d", id)
	}

	return fmt.Sprintf("%d", res.Data.Media.ID), nil
}

func (s *trackerService) fetchAniListMediaBySearch(ctx context.Context, cleanTitle string) (string, error) {
	query := `query ($search: String) {
		Media (search: $search, type: MANGA) {
			id
			title {
				english
				romaji
			}
		}
	}`

	payload := map[string]any{
		"query": query,
		"variables": map[string]any{
			"search": cleanTitle,
		},
	}

	bodyBytes, err := jsonx.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://graphql.anilist.co", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("AniList search failed with status %d", resp.StatusCode)
	}

	var res struct {
		Data struct {
			Media struct {
				ID int64 `json:"id"`
			} `json:"Media"`
		} `json:"data"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := jsonx.Unmarshal(respBody, &res); err != nil {
		return "", err
	}

	if res.Data.Media.ID == 0 {
		return "", fmt.Errorf("no AniList manga entry found for title '%s'", cleanTitle)
	}

	return fmt.Sprintf("%d", res.Data.Media.ID), nil
}

func (s *trackerService) GetOrMapBookTrackerID(ctx context.Context, bookID string, title string, provider string) (string, error) {
	mapping, err := s.repo.GetBookTrackerMapping(ctx, bookID, provider)
	if err == nil && mapping != nil && mapping.ExternalSeriesID != "" {
		return mapping.ExternalSeriesID, nil
	}

	if provider == "anilist" {
		externalID, err := s.SearchAniListMediaID(ctx, title)
		if err == nil && externalID != "" {
			_, _ = s.repo.UpsertBookTrackerMapping(ctx, bookID, provider, externalID)
			return externalID, nil
		}
	}

	return "", fmt.Errorf("tracker mapping for book ID %s not found", bookID)
}

func (s *trackerService) SyncAniListProgress(ctx context.Context, userID string, mediaID string, progress int) error {
	tracker, err := s.repo.GetUserTracker(ctx, userID, "anilist")
	if err != nil || tracker == nil {
		return apperrors.New(apperrors.ErrNotFound, "AniList integration not connected for user")
	}

	query := `mutation ($mediaId: Int, $progress: Int) {
		SaveMediaListEntry (mediaId: $mediaId, progress: $progress) {
			id
			status
			progress
		}
	}`

	variables := map[string]any{
		"mediaId":  mediaID,
		"progress": progress,
	}

	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}

	bodyBytes, err := jsonx.Marshal(payload)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "failed to serialize AniList payload")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://graphql.anilist.co", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tracker.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("AniList request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("AniList API returned error status %d: %s", resp.StatusCode, string(respBody)))
	}

	return nil
}

func (s *trackerService) SyncMyAnimeListProgress(ctx context.Context, userID string, mangaID string, chaptersRead int) error {
	tracker, err := s.repo.GetUserTracker(ctx, userID, "myanimelist")
	if err != nil || tracker == nil {
		return apperrors.New(apperrors.ErrNotFound, "MyAnimeList integration not connected for user")
	}

	url := fmt.Sprintf("https://api.myanimelist.net/v2/manga/%s/my_list_status", mangaID)
	data := fmt.Sprintf("num_chapters_read=%d", chaptersRead)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBufferString(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+tracker.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("MyAnimeList request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("MyAnimeList API returned error status %d: %s", resp.StatusCode, string(respBody)))
	}

	return nil
}

func (s *trackerService) SaveUserTracker(ctx context.Context, userID string, provider string, accessToken string) error {
	_, err := s.repo.UpsertUserTracker(ctx, userID, provider, accessToken)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("failed to save tracker for user: %v", err))
	}
	return nil
}

func (s *trackerService) SaveBookMapping(ctx context.Context, bookID string, provider string, externalSeriesID string) error {
	_, err := s.repo.UpsertBookTrackerMapping(ctx, bookID, provider, externalSeriesID)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("failed to save book tracker mapping: %v", err))
	}
	return nil
}
