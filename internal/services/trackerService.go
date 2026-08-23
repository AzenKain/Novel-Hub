package services

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
)

const anilistUserAgent = "NovelHub/1.0 (+https://github.com/novelhub)"

type TrackerService interface {
	SyncAniListProgress(ctx context.Context, userID string, mediaID string, progress int) error
	SyncMyAnimeListProgress(ctx context.Context, userID string, mangaID string, chaptersRead int) error
	SearchAniListMedia(ctx context.Context, title string) ([]response.TrackerSearchResultResponse, error)
	GetOrMapBookTrackerID(ctx context.Context, userID string, bookID string, title string, provider string) (string, error)
	SaveUserTracker(ctx context.Context, userID string, provider string, accessToken string) error
	GetUserTrackerConnections(ctx context.Context, userID string) ([]response.TrackerConnectionResponse, error)
	SaveBookMapping(ctx context.Context, userID string, bookID string, provider string, externalSeriesID string) error
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

func (s *trackerService) SearchAniListMedia(ctx context.Context, title string) ([]response.TrackerSearchResultResponse, error) {
	cleanTitle := strings.TrimSpace(title)
	if len(cleanTitle) > 100 {
		cleanTitle = cleanTitle[:100]
	}
	if cleanTitle == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	if id, err := strconv.ParseInt(cleanTitle, 10, 64); err == nil && id > 0 {
		if result, err := s.fetchAniListMediaByID(ctx, id); err == nil && result != nil {
			return []response.TrackerSearchResultResponse{*result}, nil
		}
	}

	// ponytail: two sequential calls (manga + anime) instead of one aliased query;
	// AniList has no combined "search across types" field, and this keeps each
	// GraphQL query simple. Revisit if AniList adds a type-agnostic search.
	results := make([]response.TrackerSearchResultResponse, 0, 10)
	for _, mediaType := range []string{"MANGA", "ANIME"} {
		found, err := s.fetchAniListMediaBySearch(ctx, cleanTitle, mediaType)
		if err != nil {
			continue
		}
		results = append(results, found...)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no AniList entry found for title '%s'", cleanTitle)
	}
	return results, nil
}

func (s *trackerService) fetchAniListMediaByID(ctx context.Context, id int64) (*response.TrackerSearchResultResponse, error) {
	query := `query ($id: Int) {
		Media (id: $id) {
			id
			type
			title {
				english
				romaji
			}
		}
	}`

	payload := map[string]any{
		"query": query,
		"variables": map[string]any{
			"id": id,
		},
	}

	var res struct {
		Data struct {
			Media struct {
				ID    int64  `json:"id"`
				Type  string `json:"type"`
				Title struct {
					English string `json:"english"`
					Romaji  string `json:"romaji"`
				} `json:"title"`
			} `json:"Media"`
		} `json:"data"`
	}

	if err := s.doAniListRequest(ctx, payload, "", &res); err != nil {
		return nil, err
	}

	if res.Data.Media.ID == 0 {
		return nil, fmt.Errorf("no AniList entry found for ID %d", id)
	}

	return &response.TrackerSearchResultResponse{
		ExternalSeriesID: fmt.Sprintf("%d", res.Data.Media.ID),
		TitleEnglish:     res.Data.Media.Title.English,
		TitleRomaji:      res.Data.Media.Title.Romaji,
		MediaType:        res.Data.Media.Type,
	}, nil
}

func (s *trackerService) fetchAniListMediaBySearch(ctx context.Context, cleanTitle string, mediaType string) ([]response.TrackerSearchResultResponse, error) {
	query := `query ($search: String, $type: MediaType) {
		Page (perPage: 10) {
			media (search: $search, type: $type, sort: SEARCH_MATCH) {
				id
				type
				title {
					english
					romaji
				}
			}
		}
	}`

	payload := map[string]any{
		"query": query,
		"variables": map[string]any{
			"search": cleanTitle,
			"type":   mediaType,
		},
	}

	var res struct {
		Data struct {
			Page struct {
				Media []struct {
					ID    int64  `json:"id"`
					Type  string `json:"type"`
					Title struct {
						English string `json:"english"`
						Romaji  string `json:"romaji"`
					} `json:"title"`
				} `json:"media"`
			} `json:"Page"`
		} `json:"data"`
	}

	if err := s.doAniListRequest(ctx, payload, "", &res); err != nil {
		return nil, err
	}

	results := make([]response.TrackerSearchResultResponse, 0, len(res.Data.Page.Media))
	for _, m := range res.Data.Page.Media {
		results = append(results, response.TrackerSearchResultResponse{
			ExternalSeriesID: fmt.Sprintf("%d", m.ID),
			TitleEnglish:     m.Title.English,
			TitleRomaji:      m.Title.Romaji,
			MediaType:        m.Type,
		})
	}
	return results, nil
}

// doAniListRequest posts a GraphQL payload to AniList and decodes the response into out.
// Pass a non-empty bearerToken to authenticate a mutation.
func (s *trackerService) doAniListRequest(ctx context.Context, payload map[string]any, bearerToken string, out any) error {
	bodyBytes, err := jsonx.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://graphql.anilist.co", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", anilistUserAgent)
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("AniList request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return jsonx.Unmarshal(respBody, out)
}

// The mapping is scoped to userID: the sync that follows writes to that user's own tracker
// account, so which external series a book points at is per reader, not per instance.
func (s *trackerService) GetOrMapBookTrackerID(ctx context.Context, userID string, bookID string, title string, provider string) (string, error) {
	mapping, err := s.repo.GetBookTrackerMapping(ctx, userID, bookID, provider)
	if err == nil && mapping != nil && mapping.ExternalSeriesID != "" {
		return mapping.ExternalSeriesID, nil
	}

	if provider == "anilist" {
		results, searchErr := s.SearchAniListMedia(ctx, title)
		if searchErr != nil {
			return "", searchErr
		}
		if len(results) > 0 {
			externalID := results[0].ExternalSeriesID
			_, _ = s.repo.UpsertBookTrackerMapping(ctx, userID, bookID, provider, externalID)
			return externalID, nil
		}
	}

	return "", apperrors.New(apperrors.ErrNotFound, "No tracker entry found for this book")
}

func (s *trackerService) SyncAniListProgress(ctx context.Context, userID string, mediaID string, progress int) error {
	tracker, err := s.repo.GetUserTracker(ctx, userID, "anilist")
	if err != nil || tracker == nil {
		return apperrors.New(apperrors.ErrNotFound, "AniList integration not connected for user")
	}

	// AniList's GraphQL schema expects mediaId as Int; mediaID here is the string form
	// stored in book_tracker_mappings, so it must be converted before building the payload.
	mediaIDInt, err := strconv.Atoi(mediaID)
	if err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid AniList media ID")
	}

	query := `mutation ($mediaId: Int, $progress: Int) {
		SaveMediaListEntry (mediaId: $mediaId, progress: $progress) {
			id
			status
			progress
		}
	}`

	payload := map[string]any{
		"query": query,
		"variables": map[string]any{
			"mediaId":  mediaIDInt,
			"progress": progress,
		},
	}

	var res any
	if err := s.doAniListRequest(ctx, payload, tracker.AccessToken, &res); err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("AniList sync failed: %v", err))
	}

	return nil
}

func (s *trackerService) SyncMyAnimeListProgress(ctx context.Context, userID string, mangaID string, chaptersRead int) error {
	if _, err := strconv.Atoi(mangaID); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid MyAnimeList manga ID")
	}

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
	_, err := s.repo.UpsertUserTracker(ctx, userID, provider, accessToken, nil, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("failed to save tracker for user: %v", err))
	}
	return nil
}

// Known user-connectable trackers. Tokens are stored AES-encrypted by the
// repository; this surface never returns them, only per-provider state.
var trackerConnectionProviders = []string{"anilist", "readwise", "hardcover"}

func (s *trackerService) GetUserTrackerConnections(ctx context.Context, userID string) ([]response.TrackerConnectionResponse, error) {
	out := make([]response.TrackerConnectionResponse, 0, len(trackerConnectionProviders))
	for _, provider := range trackerConnectionProviders {
		conn := response.TrackerConnectionResponse{Provider: provider, Connected: false}
		if tracker, err := s.repo.GetUserTracker(ctx, userID, provider); err == nil && tracker != nil {
			conn.Connected = true
			conn.ExpiresAt = tracker.ExpiresAt
			conn.UpdatedAt = &tracker.UpdatedAt
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.New(apperrors.ErrInternalError, "failed to load tracker connections")
		}
		out = append(out, conn)
	}
	return out, nil
}

func (s *trackerService) SaveBookMapping(ctx context.Context, userID string, bookID string, provider string, externalSeriesID string) error {
	_, err := s.repo.UpsertBookTrackerMapping(ctx, userID, bookID, provider, externalSeriesID)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("failed to save book tracker mapping: %v", err))
	}
	return nil
}
