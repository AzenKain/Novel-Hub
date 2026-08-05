package services

import (
	"bufio"
	"context"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func newOTPTestAuthService(t *testing.T) (AuthService, SettingsService, *OTPStore) {
	t.Helper()
	t.Setenv("JWT_SECRET", "otp-test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "otp-test-refresh-secret")
	t.Setenv("DB_ENCRYPTION_KEY", "otp-test-encryption-key")
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "otp.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	settingsRepo := repositories.NewSettingsRepository(db, c)
	settings := NewSettingsService(settingsRepo, database.NewTxManager(db))
	if err := settings.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	store := NewOTPStore(c)
	svc := NewAuthService(
		repositories.NewUserRepository(db, c),
		repositories.NewRoleRepository(db, c),
		database.NewTxManager(db),
		settingsRepo,
		settings,
		store,
	)
	return svc, settings, store
}

func TestOTPStoreVerifyIsSingleUseAndBounded(t *testing.T) {
	store := NewOTPStore(cache.NewRamCache())
	ctx := context.Background()

	code, err := store.Issue(ctx, OTPPurposeEmailVerify, "reader@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("code = %q, want 6 digits", code)
	}

	if _, err := store.Issue(ctx, OTPPurposeEmailVerify, "reader@example.com"); err == nil {
		t.Fatal("cooldown did not block a second request")
	}

	if _, err := store.Verify(ctx, OTPPurposePasswordReset, "reader@example.com", code); err == nil {
		t.Fatal("a code minted for email_verify was accepted for password_reset")
	}

	ticket, err := store.Verify(ctx, OTPPurposeEmailVerify, "reader@example.com", code)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(ctx, OTPPurposeEmailVerify, "reader@example.com", code); err == nil {
		t.Fatal("the same code verified twice")
	}

	if err := store.Consume(ctx, OTPPurposeEmailVerify, "reader@example.com", ticket); err != nil {
		t.Fatal(err)
	}
	if err := store.Consume(ctx, OTPPurposeEmailVerify, "reader@example.com", ticket); err == nil {
		t.Fatal("the same ticket was consumed twice")
	}
}

func TestOTPStoreLocksOutAfterMaxAttempts(t *testing.T) {
	store := NewOTPStore(cache.NewRamCache())
	ctx := context.Background()
	code, err := store.Issue(ctx, OTPPurposePasswordReset, "victim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < constants.OTPMaxAttempts; i++ {
		if _, err := store.Verify(ctx, OTPPurposePasswordReset, "victim@example.com", wrong); err == nil {
			t.Fatalf("attempt %d: a wrong code was accepted", i)
		}
	}
	if _, err := store.Verify(ctx, OTPPurposePasswordReset, "victim@example.com", code); err == nil {
		t.Fatal("the correct code still worked after the attempt limit was hit")
	}
}

func TestRegisterRequiresVerifiedTicketWhenEnabled(t *testing.T) {
	svc, settings, store := newOTPTestAuthService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, &request.RegisterDto{
		Email:    "first@example.com",
		Password: "Sup3rSecret!",
	}); err != nil {
		t.Fatalf("registration should not need a ticket while verification is off: %v", err)
	}

	if _, err := settings.UpdateSettings(ctx, map[string]any{"auth.require_email_verify": true}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Register(ctx, &request.RegisterDto{
		Email:    "second@example.com",
		Password: "Sup3rSecret!",
	}); err == nil {
		t.Fatal("registration without a ticket succeeded while verification was required")
	}

	if _, err := svc.Register(ctx, &request.RegisterDto{
		Email:     "second@example.com",
		Password:  "Sup3rSecret!",
		OTPTicket: "0192aaaa-0000-7000-8000-00000000dead",
	}); err == nil {
		t.Fatal("a forged ticket was accepted")
	}

	code, err := store.Issue(ctx, OTPPurposeEmailVerify, "second@example.com")
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := store.Verify(ctx, OTPPurposeEmailVerify, "second@example.com", code)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, &request.RegisterDto{
		Email:     "second@example.com",
		Password:  "Sup3rSecret!",
		OTPTicket: ticket,
	}); err != nil {
		t.Fatalf("a verified ticket was refused: %v", err)
	}

	if _, err := svc.Register(ctx, &request.RegisterDto{
		Email:     "third@example.com",
		Password:  "Sup3rSecret!",
		OTPTicket: ticket,
	}); err == nil {
		t.Fatal("one ticket registered two accounts")
	}
}

func TestOTPEndpointsRefusedWhileFeaturesDisabled(t *testing.T) {
	svc, settings, _ := newOTPTestAuthService(t)
	ctx := context.Background()

	if _, err := svc.RequestOTP(ctx, &request.RequestOTPDto{
		Email:   "reader@example.com",
		Purpose: "email_verify",
	}); err == nil {
		t.Fatal("OTP request worked while email verification was disabled")
	}
	if _, err := svc.RequestOTP(ctx, &request.RequestOTPDto{
		Email:   "reader@example.com",
		Purpose: "password_reset",
	}); err == nil {
		t.Fatal("OTP request worked while password reset was disabled")
	}
	if err := svc.ResetPasswordWithOTP(ctx, &request.ResetPasswordWithOTPDto{
		Email:       "reader@example.com",
		OTPTicket:   "0192aaaa-0000-7000-8000-00000000dead",
		NewPassword: "Sup3rSecret!",
	}); err == nil {
		t.Fatal("password reset worked while the feature was disabled")
	}

	if _, err := settings.UpdateSettings(ctx, map[string]any{"auth.password_reset_enabled": true}); err != nil {
		t.Fatal(err)
	}
	_, unknownErr := svc.RequestOTP(ctx, &request.RequestOTPDto{
		Email:   "nobody@example.com",
		Purpose: "password_reset",
	})
	_, knownErr := svc.RequestOTP(ctx, &request.RequestOTPDto{
		Email:   "owner@example.com",
		Purpose: "password_reset",
	})
	if unknownErr == nil || knownErr == nil {
		t.Fatal("OTP request succeeded without any mail server configured")
	}
	if unknownErr.Error() != knownErr.Error() {
		t.Fatalf("errors differ by address, revealing membership: unknown=%q known=%q", unknownErr, knownErr)
	}
}

func TestRequestOTPDoesNotRevealAccountExistence(t *testing.T) {
	svc, settings, _ := newOTPTestAuthService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, &request.RegisterDto{
		Email:    "taken@example.com",
		Password: "Sup3rSecret!",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := settings.UpdateSettings(ctx, map[string]any{"auth.require_email_verify": true}); err != nil {
		t.Fatal(err)
	}

	port, received := captureOTPMail(t)
	if _, err := settings.UpdateSettings(ctx, map[string]any{
		"smtp.enabled":                true,
		"smtp.host":                   "localhost",
		"smtp.port":                   port,
		"smtp.from_email":             "library@example.com",
		"smtp.tls_mode":               "none",
		"smtp.allow_private_networks": true,
	}); err != nil {
		t.Fatal(err)
	}

	taken, err := svc.RequestOTP(ctx, &request.RequestOTPDto{
		Email:   "taken@example.com",
		Purpose: "email_verify",
	})
	if err != nil {
		t.Fatalf("existing address leaked through an error: %v", err)
	}
	fresh, err := svc.RequestOTP(ctx, &request.RequestOTPDto{
		Email:   "brand-new@example.com",
		Purpose: "email_verify",
	})
	if err != nil {
		t.Fatalf("fresh address failed: %v", err)
	}
	if *taken != *fresh {
		t.Fatalf("responses differ, so the endpoint reveals membership: taken=%#v fresh=%#v", taken, fresh)
	}
	if taken.ExpiresInSeconds <= 0 || taken.CooldownSeconds <= 0 {
		t.Fatalf("response should carry timings the UI can show: %#v", taken)
	}

	select {
	case message := <-received:
		if strings.Contains(message, "taken@example.com") {
			t.Errorf("a verification code was mailed to an address that already has an account:\n%s", message)
		}
		if !strings.Contains(message, "brand-new@example.com") {
			t.Errorf("the mailed code did not go to the fresh address:\n%s", message)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no code was mailed for the fresh address")
	}
}

func TestResetPasswordWithOTPRevokesSessions(t *testing.T) {
	svc, settings, store := newOTPTestAuthService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, &request.RegisterDto{
		Email:    "owner@example.com",
		Password: "Sup3rSecret!",
	}); err != nil {
		t.Fatal(err)
	}
	tokens, err := svc.Signin(ctx, &request.SignInDto{Email: "owner@example.com", Password: "Sup3rSecret!"})
	if err != nil || tokens.RefreshToken == "" {
		t.Fatalf("signin failed: %v", err)
	}

	if _, err := settings.UpdateSettings(ctx, map[string]any{"auth.password_reset_enabled": true}); err != nil {
		t.Fatal(err)
	}
	code, err := store.Issue(ctx, OTPPurposePasswordReset, "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := store.Verify(ctx, OTPPurposePasswordReset, "owner@example.com", code)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetPasswordWithOTP(ctx, &request.ResetPasswordWithOTPDto{
		Email:       "owner@example.com",
		OTPTicket:   ticket,
		NewPassword: "Even-M0re-Secret!",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Signin(ctx, &request.SignInDto{Email: "owner@example.com", Password: "Sup3rSecret!"}); err == nil {
		t.Fatal("the old password still signs in")
	}
	if _, err := svc.Signin(ctx, &request.SignInDto{Email: "owner@example.com", Password: "Even-M0re-Secret!"}); err != nil {
		t.Fatalf("the new password does not sign in: %v", err)
	}
	claims, err := svc.ValidateCredentials(ctx, &request.SignInDto{Email: "owner@example.com", Password: "Even-M0re-Secret!"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RefreshToken(ctx, claims.UId, tokens.RefreshToken); err == nil {
		t.Fatal("a pre-reset refresh token still works")
	}
}

func captureOTPMail(t *testing.T) (int, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	received := make(chan string, 4)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
				reader := bufio.NewReader(conn)
				write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }
				write("220 capture ESMTP")
				var body strings.Builder
				inData := false
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if inData {
						if strings.TrimRight(line, "\r\n") == "." {
							write("250 queued")
							received <- body.String()
							body.Reset()
							inData = false
							continue
						}
						body.WriteString(line)
						continue
					}
					switch command := strings.ToUpper(strings.TrimSpace(line)); {
					case strings.HasPrefix(command, "EHLO"):
						write("250 capture")
					case strings.HasPrefix(command, "RCPT TO"):
						body.WriteString(line)
						write("250 ok")
					case strings.HasPrefix(command, "DATA"):
						write("354 send it")
						inData = true
					case strings.HasPrefix(command, "QUIT"):
						write("221 bye")
						return
					default:
						write("250 ok")
					}
				}
			}()
		}
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return number, received
}
