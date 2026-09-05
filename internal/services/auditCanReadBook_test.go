package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/mailer"
)

type auditStubSettings struct{}

func (a *auditStubSettings) Reload(ctx context.Context) error { return nil }
func (a *auditStubSettings) Public(ctx context.Context) (*models.PublicSettings, error) {
	return &models.PublicSettings{}, nil
}
func (a *auditStubSettings) Admin(ctx context.Context) (*models.AdminSettings, error) {
	return &models.AdminSettings{}, nil
}
func (a *auditStubSettings) Limits() models.RuntimeLimits { return models.RuntimeLimits{} }
func (a *auditStubSettings) ServerURL() string            { return "" }
func (a *auditStubSettings) UpdateSettings(ctx context.Context, settings map[string]any) (*models.AdminSettings, error) {
	return nil, nil
}
func (a *auditStubSettings) GuestAllows(libraryID string) bool { return false }
func (a *auditStubSettings) SetupRequired(ctx context.Context) bool {
	return false
}
func (a *auditStubSettings) SaveAsset(ctx context.Context, target string, fileData []byte, fileName string, urlStr string) (string, error) {
	return "", nil
}
func (a *auditStubSettings) SMTP(ctx context.Context) (mailer.SMTPConfig, error) {
	return mailer.SMTPConfig{}, nil
}
func (a *auditStubSettings) TestSMTP(ctx context.Context, dto *request.SMTPTestDto) error { return nil }
func (a *auditStubSettings) OAuthProviderConfig(ctx context.Context, provider string) (*models.OAuthProviderConfig, error) {
	return nil, nil
}
func (a *auditStubSettings) HardcoverConfig(ctx context.Context) (*models.HardcoverConfig, error) {
	return nil, nil
}

// TestAuditCanReadBookIgnoresAgeRating proves task T0.3: CanReadBook does not enforce age rating / kids mode.
func TestAuditCanReadBookIgnoresAgeRating(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "audit_canread.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1', 'Audit Lib')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, email, full_name, is_kids_mode, max_allowed_age_rating) VALUES ('u-kids', 'kid@example.com', 'Kid', 1, 'G')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, age_rating) VALUES ('b-r18', 'lib-1', 'Mature Title', 'R18+')`); err != nil {
		t.Fatal(err)
	}

	var userRoleID string
	if err := db.QueryRow(`SELECT id FROM roles WHERE name = 'USER'`).Scan(&userRoleID); err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	permissions := NewPermissionCache(repositories.NewRoleRepository(db, c))
	if err := permissions.Reload(ctx); err != nil {
		t.Fatal(err)
	}

	svc := &bookService{settings: &auditStubSettings{}, permissions: permissions}
	claims := &response.JWTClaims{UId: "u-kids", RoleIDs: []string{userRoleID}}

	gBook := &models.BookEntity{ID: "b-r18", LibraryID: "lib-1", AgeRating: constants.AgeRatingG}
	if !svc.CanReadBook(ctx, gBook, claims) {
		t.Fatalf("setup broken: CanReadBook denied a G book for a USER-role holder; cannot probe the age-rating gap")
	}

	r18Book := &models.BookEntity{ID: "b-r18", LibraryID: "lib-1", AgeRating: constants.AgeRatingR18}
	if !svc.CanReadBook(ctx, r18Book, claims) {
		t.Fatalf("unexpected: CanReadBook returned false — age-rating enforcement may have been added; the audit claim no longer holds")
	}
}
