package services

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newMailUserSvc(t *testing.T) (UserService, SettingsService) {
	t.Helper()
	t.Setenv("DB_ENCRYPTION_KEY", "user-email-test-key")
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "user-email.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (id, email, auth_provider, token_version) VALUES
			('01920000-0000-7000-8000-0000000000a1','reader@example.com','LOCAL',1),
			('01920000-0000-7000-8000-0000000000a2','gone@example.com','LOCAL',1)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE users SET is_deleted = 1 WHERE id = '01920000-0000-7000-8000-0000000000a2'`); err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	settings := NewSettingsService(repositories.NewSettingsRepository(db, c), database.NewTxManager(db))
	if err := settings.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	svc := NewUserService(
		repositories.NewUserRepository(db, c),
		repositories.NewRoleRepository(db, c),
		repositories.NewSettingsRepository(db, c),
		database.NewTxManager(db),
		NewPermissionCache(repositories.NewRoleRepository(db, c)),
		settings,
	)
	return svc, settings
}

func TestSendUserEmailUsesTheStoredAddress(t *testing.T) {
	svc, settings := newMailUserSvc(t)
	ctx := context.Background()
	port, received := captureOTPMail(t)
	enableCaptureSMTP(t, settings, port)

	if err := svc.SendEmail(ctx, "01920000-0000-7000-8000-0000000000a1", &request.SendUserEmailDto{
		Subject: "Library maintenance",
		Body:    "The server will restart tonight.",
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case message := <-received:
		for _, want := range []string{"reader@example.com", "Library maintenance", "The server will restart tonight."} {
			if !strings.Contains(message, want) {
				t.Errorf("delivered mail is missing %q:\n%s", want, message)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no mail was delivered to the target user")
	}
}

func TestSendUserEmailRefusesUnknownDeletedAndUnconfigured(t *testing.T) {
	svc, settings := newMailUserSvc(t)
	ctx := context.Background()
	dto := &request.SendUserEmailDto{Subject: "Hi", Body: "Hello"}

	if err := svc.SendEmail(ctx, "01920000-0000-7000-8000-0000000000a1", dto); err == nil {
		t.Fatal("mail was reported sent while SMTP was disabled")
	}

	port, received := captureOTPMail(t)
	enableCaptureSMTP(t, settings, port)

	if err := svc.SendEmail(ctx, "01920000-0000-7000-8000-0000000000a2", dto); err == nil {
		t.Fatal("a soft-deleted account was emailed")
	}
	if err := svc.SendEmail(ctx, "01920000-0000-7000-8000-00000000dead", dto); err == nil {
		t.Fatal("an unknown user id was accepted")
	}
	if err := svc.SendEmail(ctx, "not-a-uuid", dto); err == nil {
		t.Fatal("a malformed user id was accepted")
	}

	select {
	case message := <-received:
		t.Fatalf("a refused request still delivered mail:\n%s", message)
	case <-time.After(500 * time.Millisecond):
	}
}
