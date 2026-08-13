package services

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/netx"
	"novelhub/pkg/worker"
)

// --- stub PodcastRepository (in-memory) ---

type stubPodcastRepo struct {
	mu       sync.Mutex
	podcasts map[string]*models.PodcastEntity
	guids    map[string][]string
	episodes map[string]*models.PodcastEpisodeEntity
}

func newStubPodcastRepo() *stubPodcastRepo {
	return &stubPodcastRepo{
		podcasts: map[string]*models.PodcastEntity{},
		guids:    map[string][]string{},
		episodes: map[string]*models.PodcastEpisodeEntity{},
	}
}

func (s *stubPodcastRepo) WithTx(tx *sql.Tx) repositories.PodcastRepository { return s }

func (s *stubPodcastRepo) CreatePodcast(ctx context.Context, id, libraryID, feedURL, title string, description, coverURL, author *string) (*models.PodcastEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := &models.PodcastEntity{ID: id, LibraryID: libraryID, FeedURL: feedURL, Title: title, Description: description, CoverURL: coverURL, Author: author}
	now := time.Now()
	p.CreatedAt, p.UpdatedAt = &now, &now
	s.podcasts[id] = p
	return p, nil
}

func (s *stubPodcastRepo) GetPodcast(ctx context.Context, id string) (*models.PodcastEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.podcasts[id], nil
}

func (s *stubPodcastRepo) GetPodcastByFeedURL(ctx context.Context, feedURL string) (*models.PodcastEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.podcasts {
		if p.FeedURL == feedURL {
			return p, nil
		}
	}
	return nil, nil
}

func (s *stubPodcastRepo) ListPodcastIDs(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.podcasts))
	for id := range s.podcasts {
		out = append(out, id)
	}
	return out, nil
}

func (s *stubPodcastRepo) ListPodcastsWithCounts(ctx context.Context) ([]*models.PodcastWithCountEntity, error) {
	return nil, nil
}

func (s *stubPodcastRepo) UpdatePodcast(ctx context.Context, id string, title string, description, coverURL, author *string, autoDownload bool, lastCheckedAt *time.Time) (*models.PodcastEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.podcasts[id]
	if p == nil {
		return nil, nil
	}
	p.Title, p.Description, p.CoverURL, p.Author, p.AutoDownload, p.LastCheckedAt = title, description, coverURL, author, autoDownload, lastCheckedAt
	return p, nil
}

func (s *stubPodcastRepo) DeletePodcast(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.podcasts, id)
	return nil
}

func (s *stubPodcastRepo) UpsertEpisode(ctx context.Context, id, podcastID, guid, title string, description *string, audioURL string, durationSec *int64, publishedAt *time.Time) (*models.PodcastEpisodeEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep := &models.PodcastEpisodeEntity{ID: id, PodcastID: podcastID, GUID: guid, Title: title, Description: description, AudioURL: audioURL, DurationSec: durationSec, PublishedAt: publishedAt}
	now := time.Now()
	ep.CreatedAt, ep.UpdatedAt = &now, &now
	s.episodes[id] = ep
	for _, g := range s.guids[podcastID] {
		if g == guid {
			return ep, nil
		}
	}
	s.guids[podcastID] = append(s.guids[podcastID], guid)
	return ep, nil
}

func (s *stubPodcastRepo) ListEpisodeGuids(ctx context.Context, podcastID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.guids[podcastID]...), nil
}

func (s *stubPodcastRepo) ListEpisodes(ctx context.Context, podcastID string) ([]*models.PodcastEpisodeEntity, error) {
	return nil, nil
}

func (s *stubPodcastRepo) GetEpisode(ctx context.Context, id string) (*models.PodcastEpisodeEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.episodes[id], nil
}

func (s *stubPodcastRepo) MarkEpisodeDownloaded(ctx context.Context, episodeID string, bookID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ep := s.episodes[episodeID]; ep != nil {
		ep.Downloaded = true
		ep.BookID = &bookID
	}
	return nil
}

// --- RSS parsing ---

func TestParseFeed(t *testing.T) {
	feed := `<?xml version="1.0"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
<channel>
  <title>Tech Talk</title>
  <description>Weekly tech show</description>
  <itunes:author>Jane Doe</itunes:author>
  <itunes:image href="https://example.com/cover.jpg"/>
  <item>
    <title>Plain title</title>
    <guid>ep-1</guid>
    <enclosure url="https://example.com/ep1.mp3" type="audio/mpeg"/>
    <itunes:duration>01:02:03</itunes:duration>
    <pubDate>Mon, 02 Jan 2023 15:04:05 GMT</pubDate>
  </item>
  <item>
    <itunes:title>Itunes-titled episode</itunes:title>
    <guid>ep-2</guid>
    <enclosure url="https://example.com/ep2.m4a"/>
    <itunes:duration>37:45</itunes:duration>
  </item>
  <item>
    <guid>ep-3</guid>
    <itunes:duration>1800</itunes:duration>
  </item>
  <item>
    <title>No guid</title>
    <enclosure url="https://example.com/ep4.mp3"/>
  </item>
</channel>
</rss>`

	pod, err := parseFeed([]byte(feed))
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if pod.Title != "Tech Talk" || pod.Author != "Jane Doe" || pod.CoverURL != "https://example.com/cover.jpg" {
		t.Fatalf("podcast meta = %+v", pod)
	}
	if len(pod.Episodes) != 2 { // ep-3 (no enclosure) and no-guid item skipped
		t.Fatalf("got %d episodes, want 2 (unusable items skipped)", len(pod.Episodes))
	}
	ep1, ep2 := pod.Episodes[0], pod.Episodes[1]
	if ep1.Title != "Plain title" || ep1.DurationSec == nil || *ep1.DurationSec != 3723 {
		t.Fatalf("ep1 = %+v, want title 'Plain title' duration 3723", ep1)
	}
	if ep1.PublishedAt == nil || ep1.PublishedAt.Year() != 2023 {
		t.Fatalf("ep1 PublishedAt = %v", ep1.PublishedAt)
	}
	if ep2.Title != "Itunes-titled episode" || ep2.DurationSec == nil || *ep2.DurationSec != 2265 {
		t.Fatalf("ep2 = %+v, want itunes title and duration 2265", ep2)
	}
}

func TestParseFeedRejectsGarbage(t *testing.T) {
	if _, err := parseFeed([]byte("not xml")); err == nil {
		t.Fatal("expected error for non-XML input")
	}
	if _, err := parseFeed([]byte(`<rss><channel></channel></rss>`)); err == nil {
		t.Fatal("expected error for channel without title")
	}
}

func TestParseFeedImageFallback(t *testing.T) {
	feed := `<rss><channel><title>T</title><image><url>https://example.com/img.png</url></image><item><guid>g</guid><enclosure url="https://e/a.mp3"/></item></channel></rss>`
	pod, err := parseFeed([]byte(feed))
	if err != nil {
		t.Fatal(err)
	}
	if pod.CoverURL != "https://example.com/img.png" {
		t.Fatalf("cover = %q, want image>url fallback", pod.CoverURL)
	}
}

// --- refresh: dedup by guid + auto-download enqueues only new episodes ---

func feedHandler(_ *testing.T, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
}

func TestRefreshPodcastDedupByGuidAndAutoDownload(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	svr := feedHandler(t, `<?xml version="1.0"?><rss version="2.0"><channel><title>Show</title><item><title>Ep A</title><guid>a</guid><enclosure url="https://x/a.mp3"/></item><item><title>Ep B</title><guid>b</guid><enclosure url="https://x/b.mp3"/></item></channel></rss>`)
	defer svr.Close()

	repo := newStubPodcastRepo()
	now := time.Now()
	_, _ = repo.CreatePodcast(context.Background(), "pod-1", "lib-1", svr.URL, "Show", nil, nil, nil)
	repo.podcasts["pod-1"].AutoDownload = true
	repo.podcasts["pod-1"].CreatedAt = &now

	q := worker.NewQueue(2)
	defer q.Stop()
	var enqueued []string
	var mu sync.Mutex
	q.RegisterHandler(podcastDownloadJobType, func(_ context.Context, _ string, payload string) error {
		mu.Lock()
		enqueued = append(enqueued, payload)
		mu.Unlock()
		return nil
	})
	q.Start()

	s := NewPodcastService(repo, nil, nil, nil, q)

	if err := s.RefreshPodcast(context.Background(), "pod-1"); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	guids, err := repo.ListEpisodeGuids(context.Background(), "pod-1")
	if err != nil || len(guids) != 2 {
		t.Fatalf("after first refresh: %d guids (want 2), err=%v", len(guids), err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(enqueued)
		mu.Unlock()
		if n == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	firstEnqueued := len(enqueued)
	mu.Unlock()
	if firstEnqueued != 2 {
		t.Fatalf("first refresh enqueued %d downloads, want 2 (new episodes)", firstEnqueued)
	}

	// second refresh: same feed, no new guids -> 0 downloads, guids unchanged
	if err := s.RefreshPodcast(context.Background(), "pod-1"); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	guids, _ = repo.ListEpisodeGuids(context.Background(), "pod-1")
	if len(guids) != 2 {
		t.Fatalf("after second refresh: %d guids, want 2 (dedup)", len(guids))
	}
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	secondEnqueued := len(enqueued)
	mu.Unlock()
	if secondEnqueued != 2 {
		t.Fatalf("second refresh enqueued %d downloads total, want still 2", secondEnqueued)
	}
}

func TestRefreshAllContinuesPastBadFeed(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	good := feedHandler(t, `<rss><channel><title>Ok</title><item><guid>g1</guid><enclosure url="https://x/a.mp3"/></item></channel></rss>`)
	defer good.Close()

	repo := newStubPodcastRepo()
	_, _ = repo.CreatePodcast(context.Background(), "bad", "lib-1", bad.URL, "Bad", nil, nil, nil)
	_, _ = repo.CreatePodcast(context.Background(), "good", "lib-1", good.URL, "Ok", nil, nil, nil)

	s := NewPodcastService(repo, nil, nil, nil, nil)
	if err := s.RefreshAllPodcasts(context.Background()); err != nil {
		t.Fatalf("RefreshAllPodcasts should swallow individual feed failures, got %v", err)
	}
	guids, _ := repo.ListEpisodeGuids(context.Background(), "good")
	if len(guids) != 1 {
		t.Fatalf("good feed not refreshed: got %d guids, want 1", len(guids))
	}
}

func TestRefreshPodcastNotFound(t *testing.T) {
	s := NewPodcastService(newStubPodcastRepo(), nil, nil, nil, nil)
	if err := s.RefreshPodcast(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for unknown podcast")
	}
}

// --- download: already-downloaded episode is a no-op ---

func TestExecutePodcastDownloadJobSkipsDownloaded(t *testing.T) {
	repo := newStubPodcastRepo()
	now := time.Now()
	downloaded := true
	ep := &models.PodcastEpisodeEntity{ID: "ep-1", PodcastID: "pod-1", GUID: "g", AudioURL: "http://localhost:1/x.mp3", Downloaded: downloaded, CreatedAt: &now, UpdatedAt: &now}
	repo.episodes["ep-1"] = ep

	s := NewPodcastService(repo, nil, nil, nil, nil)
	if err := s.ExecutePodcastDownloadJob(context.Background(), `{"podcast_id":"pod-1","episode_id":"ep-1"}`); err != nil {
		t.Fatalf("expected no-op for already-downloaded episode, got %v", err)
	}
}

func TestExecutePodcastDownloadJobRejectsBadPayload(t *testing.T) {
	s := NewPodcastService(newStubPodcastRepo(), nil, nil, nil, nil)
	if err := s.ExecutePodcastDownloadJob(context.Background(), `{not json`); err == nil {
		t.Fatal("expected error for malformed payload")
	}
}

func TestParseFeedTemplateDetection(t *testing.T) {
	templateData := []byte(`
---
layout: null
---
<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <title>{{ site.podcast.title }}</title>
  </channel>
</rss>
`)
	_, err := parseFeed(templateData)
	if err == nil {
		t.Fatal("expected error parsing uncompiled template feed, got nil")
	}
	if !strings.Contains(err.Error(), "uncompiled template") {
		t.Errorf("expected error message to contain 'uncompiled template', got: %v", err)
	}
}