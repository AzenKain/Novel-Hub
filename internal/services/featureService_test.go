package services

import (
	"context"
	"errors"
	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/mailer"

	"github.com/google/uuid"
	"testing"
	"time"
)

type sessionPermissionStub struct{ allowed bool }

func (p sessionPermissionStub) Reload(context.Context) error { return nil }
func (p sessionPermissionStub) Can(context.Context, string, string, map[string]any) bool {
	return p.allowed
}
func (p sessionPermissionStub) CanRoles([]string, []constants.RoleType, string, map[string]any) bool {
	return p.allowed
}
func (p sessionPermissionStub) IsAdmin([]string, []constants.RoleType) bool { return false }
func (p sessionPermissionStub) GetGuestPermissions() []string               { return nil }
func (p sessionPermissionStub) DescribeRoles([]string) []*models.RoleSimple { return nil }

type sessionBookRepo struct {
	repositories.BookDBRepository
	book *models.BookEntity
	err  error
}

func (r sessionBookRepo) GetBook(context.Context, string) (*models.BookEntity, error) {
	return r.book, r.err
}

type sessionFeatureRepo struct {
	repositories.FeatureRepository
	err error
	got sqlc.UpsertReadingSessionParams
}

func (r *sessionFeatureRepo) UpsertReadingSession(_ context.Context, a sqlc.UpsertReadingSessionParams) (*models.ReadingSessionEntity, error) {
	r.got = a
	return nil, r.err
}
func newSessionService(r repositories.FeatureRepository, b *models.BookEntity, be error, a bool) *featureService {
	return &featureService{repo: r, bookRepo: sessionBookRepo{book: b, err: be}, permissions: sessionPermissionStub{allowed: a}}
}
func TestRecordReadingSessionGeneratesID(t *testing.T) {
	r := &sessionFeatureRepo{}
	if e := newSessionService(r, &models.BookEntity{LibraryID: "l"}, nil, true).RecordReadingSession(context.Background(), "u", "b", 3, 7, "", &response.JWTClaims{}); e != nil {
		t.Fatal(e)
	}
	if r.got.ID == "" {
		t.Fatal("missing ID")
	}
}
func TestRecordReadingSessionInaccessibleBook(t *testing.T) {
	r := &sessionFeatureRepo{}
	e := newSessionService(r, nil, errors.New("missing"), true).RecordReadingSession(context.Background(), "u", "b", 1, 0, "", &response.JWTClaims{})
	if !errors.Is(e, apperrors.ErrForbidden) {
		t.Fatal(e)
	}
	if r.got.ID != "" {
		t.Fatal("called repository")
	}
}
func TestRecordReadingSessionRepositoryFailurePreservesCause(t *testing.T) {
	c := errors.New("NOT NULL constraint failed: reading_sessions.id")
	r := &sessionFeatureRepo{err: c}
	e := newSessionService(r, &models.BookEntity{LibraryID: "l"}, nil, true).RecordReadingSession(context.Background(), "u", "b", 1, 0, "", &response.JWTClaims{})
	if !errors.Is(e, apperrors.ErrInternalError) || !errors.Is(e, c) {
		t.Fatalf("causes not preserved: %v", e)
	}
}

func TestGetBookUserStateRequiresBookAccess(t *testing.T) {
	svc := newSessionService(&sessionFeatureRepo{}, &models.BookEntity{ID: "b", LibraryID: "l"}, nil, false)
	_, err := svc.GetBookUserState(context.Background(), "u", "b", &response.JWTClaims{UId: "u"})
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Fatalf("book the user cannot read must be forbidden, got %v", err)
	}
}

var _ repositories.BookDBRepository = sessionBookRepo{}
var _ repositories.FeatureRepository = (*sessionFeatureRepo)(nil)

type statsFeatureRepo struct {
	repositories.FeatureRepository
	heatmap       []*models.ReadingHeatmapEntity
	goal          *models.ReadingGoalEntity
	goalErr       error
	byBook        *models.ReadingStatsByBookEntity
	since         *models.ReadingStatsSinceEntity
	progress      *models.ReadingProgressEntity
	progressErr   error
	breakdown     *models.LibraryBreakdownEntity
	listening     []*models.ListeningHistoryEntity
	listeningStats *models.ListeningStatsEntity
}

func (r *statsFeatureRepo) GetReadingHeatmap(context.Context, string) ([]*models.ReadingHeatmapEntity, error) {
	return r.heatmap, nil
}
func (r *statsFeatureRepo) GetReadingGoal(context.Context, string) (*models.ReadingGoalEntity, error) {
	if r.goalErr != nil {
		return nil, r.goalErr
	}
	return r.goal, nil
}
func (r *statsFeatureRepo) GetReadingStatsByBook(context.Context, string, string) (*models.ReadingStatsByBookEntity, error) {
	return r.byBook, nil
}
func (r *statsFeatureRepo) GetReadingStatsSince(context.Context, string, time.Time) (*models.ReadingStatsSinceEntity, error) {
	return r.since, nil
}
func (r *statsFeatureRepo) GetReadingProgress(context.Context, string, string) (*models.ReadingProgressEntity, error) {
	if r.progressErr != nil {
		return nil, r.progressErr
	}
	return r.progress, nil
}
func (r *statsFeatureRepo) GetLibraryBreakdown(context.Context) (*models.LibraryBreakdownEntity, error) {
	return r.breakdown, nil
}
func (r *statsFeatureRepo) GetListeningHistory(context.Context, string) ([]*models.ListeningHistoryEntity, error) {
	return r.listening, nil
}
func (r *statsFeatureRepo) GetListeningStats(context.Context, string) (*models.ListeningStatsEntity, error) {
	return r.listeningStats, nil
}

func day(offset int) time.Time {
	return time.Now().AddDate(0, 0, offset)
}

func heatmapDay(offset int, words int64) *models.ReadingHeatmapEntity {
	return &models.ReadingHeatmapEntity{Date: day(offset), WordsRead: words, DurationSeconds: words * 60}
}

func TestComputeStreaks(t *testing.T) {
	cases := []struct {
		name             string
		active           map[string]struct{}
		current, longest int64
	}{
		{"empty", map[string]struct{}{}, 0, 0},
		{"only today", map[string]struct{}{day(0).Format("2006-01-02"): {}}, 1, 1},
		{"today and yesterday", map[string]struct{}{day(0).Format("2006-01-02"): {}, day(-1).Format("2006-01-02"): {}}, 2, 2},
		{"yesterday only counts current", map[string]struct{}{day(-1).Format("2006-01-02"): {}}, 1, 1},
		{"gap breaks current", map[string]struct{}{day(0).Format("2006-01-02"): {}, day(-2).Format("2006-01-02"): {}}, 1, 1},
		{"longest beats current", map[string]struct{}{day(0).Format("2006-01-02"): {}, day(-1).Format("2006-01-02"): {}, day(-10).Format("2006-01-02"): {}, day(-11).Format("2006-01-02"): {}, day(-12).Format("2006-01-02"): {}}, 2, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cur, longest := computeStreaks(c.active, time.Now())
			if cur != c.current || longest != c.longest {
				t.Fatalf("got current=%d longest=%d, want %d/%d", cur, longest, c.current, c.longest)
			}
		})
	}
}

func TestGetReadingStatsSummary(t *testing.T) {
	r := &statsFeatureRepo{
		heatmap: []*models.ReadingHeatmapEntity{
			heatmapDay(0, 500),
			heatmapDay(-1, 300),
			heatmapDay(-3, 200),
		},
		goal: &models.ReadingGoalEntity{TargetWordsPerDay: 2000, TargetBooksPerYear: 24},
	}
	svc := newSessionService(r, &models.BookEntity{LibraryID: "l"}, nil, true)
	res, err := svc.GetReadingStatsSummary(context.Background(), "u")
	if err != nil {
		t.Fatal(err)
	}
	if res.CurrentStreakDays != 2 || res.LongestStreakDays != 2 {
		t.Fatalf("streaks wrong: %d/%d", res.CurrentStreakDays, res.LongestStreakDays)
	}
	if res.TotalActiveDays != 3 || res.TotalWords != 1000 || res.TotalMinutes != 1000 {
		t.Fatalf("totals wrong: %+v", res)
	}
	if res.WordsToday != 500 || res.WordsTodayTarget != 2000 || res.BooksPerYearTarget != 24 {
		t.Fatalf("targets wrong: %+v", res)
	}
}

func TestGetReadingStatsSummaryDefaultGoal(t *testing.T) {
	r := &statsFeatureRepo{goalErr: apperrors.New(apperrors.ErrNotFound, "none")}
	svc := newSessionService(r, &models.BookEntity{LibraryID: "l"}, nil, true)
	res, err := svc.GetReadingStatsSummary(context.Background(), "u")
	if err != nil {
		t.Fatal(err)
	}
	if res.WordsTodayTarget != defaultTargetWordsPerDay || res.BooksPerYearTarget != defaultTargetBooksPerYear {
		t.Fatalf("defaults not applied: %+v", res)
	}
}

func TestGetReaderETAProgressMath(t *testing.T) {
	r := &statsFeatureRepo{
		byBook:   &models.ReadingStatsByBookEntity{BookID: "b", TotalDuration: 6000, TotalWords: 2500},
		progress: &models.ReadingProgressEntity{ProgressPercent: ptr(float64(50))},
	}
	svc := newSessionService(r, &models.BookEntity{ID: "b", LibraryID: "l"}, nil, true)
	res, err := svc.GetReaderETA(context.Background(), "u", "b", &response.JWTClaims{UId: "u"})
	if err != nil {
		t.Fatal(err)
	}
	if res.PaceWordsPerMin != 25 {
		t.Fatalf("pace wrong: %v", res.PaceWordsPerMin)
	}
	if res.WordsRead != 2500 || res.Percent != 50 {
		t.Fatalf("words/percent wrong: %+v", res)
	}
	if res.RemainingWords != 2500 || res.EtaMinutes != 100 {
		t.Fatalf("remaining wrong: remaining=%d eta=%d", res.RemainingWords, res.EtaMinutes)
	}
}

func TestGetReaderETAFallsBackToGlobalPace(t *testing.T) {
	r := &statsFeatureRepo{
		since: &models.ReadingStatsSinceEntity{TotalDuration: 3600, TotalWords: 600},
	}
	svc := newSessionService(r, &models.BookEntity{ID: "b", LibraryID: "l"}, nil, true)
	res, err := svc.GetReaderETA(context.Background(), "u", "b", &response.JWTClaims{UId: "u"})
	if err != nil {
		t.Fatal(err)
	}
	if res.PaceWordsPerMin != 10 {
		t.Fatalf("global pace wrong: %v", res.PaceWordsPerMin)
	}
}

func TestGetReaderETAForbiddenWithoutAccess(t *testing.T) {
	svc := newSessionService(&statsFeatureRepo{}, nil, errors.New("missing"), true)
	_, err := svc.GetReaderETA(context.Background(), "u", "b", &response.JWTClaims{UId: "u"})
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestGetLibraryBreakdownAvgSpeed(t *testing.T) {
	r := &statsFeatureRepo{
		breakdown:     &models.LibraryBreakdownEntity{},
		listeningStats: &models.ListeningStatsEntity{TotalWords: 3000, TotalDuration: 3600},
	}
	svc := newSessionService(r, nil, nil, true)
	res, err := svc.GetLibraryBreakdown(context.Background(), "u")
	if err != nil {
		t.Fatal(err)
	}
	if res.AvgSpeedWpm != 50 {
		t.Fatalf("avg speed wrong: %v", res.AvgSpeedWpm)
	}
}

func TestGetLibraryBreakdownAvgSpeedNoDuration(t *testing.T) {
	r := &statsFeatureRepo{
		breakdown:      &models.LibraryBreakdownEntity{},
		listeningStats: &models.ListeningStatsEntity{TotalWords: 500, TotalDuration: 0},
	}
	svc := newSessionService(r, nil, nil, true)
	res, err := svc.GetLibraryBreakdown(context.Background(), "u")
	if err != nil {
		t.Fatal(err)
	}
	if res.AvgSpeedWpm != 0 {
		t.Fatalf("expected 0 speed on zero duration, got %v", res.AvgSpeedWpm)
	}
}

func ptr(f float64) *float64 { return &f }

type libraryStatsSettingsStub struct {
	guestLoginRequired bool
	err                error
}

func (s libraryStatsSettingsStub) Reload(context.Context) error                { return nil }
func (s libraryStatsSettingsStub) Public(context.Context) (*models.PublicSettings, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &models.PublicSettings{GuestLoginRequired: s.guestLoginRequired}, nil
}
func (s libraryStatsSettingsStub) Admin(context.Context) (*models.AdminSettings, error) { return nil, nil }
func (s libraryStatsSettingsStub) Limits() models.RuntimeLimits                         { return models.RuntimeLimits{} }
func (s libraryStatsSettingsStub) ServerURL() string                                    { return "" }
func (s libraryStatsSettingsStub) UpdateSettings(context.Context, map[string]any) (*models.AdminSettings, error) {
	return nil, nil
}
func (s libraryStatsSettingsStub) GuestAllows(string) bool       { return !s.guestLoginRequired }
func (s libraryStatsSettingsStub) SetupRequired(context.Context) bool { return false }
func (s libraryStatsSettingsStub) SaveAsset(context.Context, string, []byte, string, string) (string, error) {
	return "", nil
}
func (s libraryStatsSettingsStub) SMTP(context.Context) (mailer.SMTPConfig, error) {
	return mailer.SMTPConfig{}, nil
}
func (s libraryStatsSettingsStub) TestSMTP(context.Context, *request.SMTPTestDto) error { return nil }
func (s libraryStatsSettingsStub) OAuthProviderConfig(context.Context, string) (*models.OAuthProviderConfig, error) {
	return nil, nil
}
func (s libraryStatsSettingsStub) HardcoverConfig(context.Context) (*models.HardcoverConfig, error) {
	return nil, nil
}

type libraryStatsRepo struct {
	sessionFeatureRepo
	called bool
}

func (r *libraryStatsRepo) GetLibraryStats(context.Context) (*models.LibraryStatsEntity, error) {
	r.called = true
	return &models.LibraryStatsEntity{TotalBooks: 5}, nil
}

func newLibraryStatsService(settings SettingsService) (*featureService, *libraryStatsRepo) {
	repo := &libraryStatsRepo{}
	return &featureService{repo: repo, settings: settings}, repo
}

func TestGetLibraryStatsAllowsGuestWhenGuestAccessEnabled(t *testing.T) {
	svc, repo := newLibraryStatsService(libraryStatsSettingsStub{})
	stats, err := svc.GetLibraryStats(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !repo.called || stats.TotalBooks != 5 {
		t.Fatalf("expected stats for guest, got called=%v stats=%+v", repo.called, stats)
	}
}

func TestGetLibraryStatsBlocksGuestWhenLoginRequired(t *testing.T) {
	svc, repo := newLibraryStatsService(libraryStatsSettingsStub{guestLoginRequired: true})
	if _, err := svc.GetLibraryStats(context.Background(), nil); apperrors.IsNotFound(err) || err == nil {
		t.Fatalf("expected auth error for guest, got %v", err)
	}
	if repo.called {
		t.Fatal("repository must not be queried when guest access is disabled")
	}
}

func TestGetLibraryStatsAllowsAuthenticatedUser(t *testing.T) {
	svc, repo := newLibraryStatsService(libraryStatsSettingsStub{guestLoginRequired: true})
	if _, err := svc.GetLibraryStats(context.Background(), &response.JWTClaims{UId: "7"}); err != nil {
		t.Fatal(err)
	}
	if !repo.called {
		t.Fatal("repository should be queried for authenticated user")
	}
}

func TestReorderRolesCommitsAtomically(t *testing.T) {
	db := auditDB(t)
	ram := cache.NewRamCache()
	roleRepo := repositories.NewRoleRepository(db, ram)
	perms := NewPermissionCache(roleRepo)
	if err := perms.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	svc := NewRoleService(roleRepo, perms, database.NewTxManager(db))

	ids := make([]string, 0, 3)
	for _, name := range []string{"alpha", "beta", "gamma"} {
		id := uuid.Must(uuid.NewV7()).String()
		role, err := roleRepo.Create(context.Background(), sqlc.CreateRoleParams{ID: id, Name: name})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, role.ID)
	}

	if err := svc.ReorderRoles(context.Background(), &request.ReorderRolesDto{RoleIDs: ids}); err != nil {
		t.Fatal(err)
	}

	// First ID in the payload gets the highest position (total*10, descending).
	want := []int64{30, 20, 10}
	for i, id := range ids {
		role, err := roleRepo.GetByID(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if role.Position != want[i] {
			t.Fatalf("role %d: want position %d, got %d", i, want[i], role.Position)
		}
	}
}

func TestReorderRolesRejectsEmptyPayload(t *testing.T) {
	db := auditDB(t)
	roleRepo := repositories.NewRoleRepository(db, cache.NewRamCache())
	perms := NewPermissionCache(roleRepo)
	svc := NewRoleService(roleRepo, perms, database.NewTxManager(db))
	if err := svc.ReorderRoles(context.Background(), &request.ReorderRolesDto{}); err == nil {
		t.Fatal("expected error for empty role_ids")
	}
}
