package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func captureSMTP(t *testing.T) (int, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
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
					inData = false
					continue
				}
				body.WriteString(line)
				continue
			}
			switch command := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(command, "EHLO"):
				write("250 capture")
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

func TestSendBookToEmailUsesConfiguredSMTP(t *testing.T) {
	t.Setenv("DB_ENCRYPTION_KEY", "smtp-e2e-test-key")
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("SQLITE_DB_PATH", filepath.Join(dataDir, "novelhub-smtp-test.db"))

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db, "../../db/schema"); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	smtpPort, received := captureSMTP(t)
	smtpSettings := map[string]string{
		"smtp.enabled":                "true",
		"smtp.host":                   `"localhost"`,
		"smtp.port":                   strconv.Itoa(smtpPort),
		"smtp.from_email":             `"library@example.com"`,
		"smtp.tls_mode":               `"none"`,
		"smtp.allow_private_networks": "true",
	}
	for key, value := range smtpSettings {
		if _, err := db.Exec(`
			INSERT INTO app_settings (key, value_json) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
		`, key, value); err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
	}

	userID := uuid.Must(uuid.NewV7()).String()
	hash, err := bcrypt.GenerateFromPassword([]byte("Sup3rSecret!"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, auth_provider) VALUES (?, ?, ?, 'LOCAL')
	`, userID, "reader@example.com", string(hash)); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	var adminRoleID string
	if err := db.QueryRow(`SELECT id FROM roles WHERE is_admin = 1 LIMIT 1`).Scan(&adminRoleID); err != nil {
		t.Fatalf("find admin role: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, userID, adminRoleID); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}

	libraryID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES (?, 'Main')`, libraryID); err != nil {
		t.Fatalf("seed library: %v", err)
	}
	bookID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO books (id, library_id, title, status) VALUES (?, ?, 'Dune', 'active')
	`, bookID, libraryID); err != nil {
		t.Fatalf("seed book: %v", err)
	}
	bookPath := filepath.Join(dataDir, "dune.epub")
	if err := os.WriteFile(bookPath, []byte("epub-bytes-for-the-attachment"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO book_files (id, book_id, format, path, size_bytes, mod_time)
		VALUES (?, ?, 'epub', ?, 29, CURRENT_TIMESTAMP)
	`, uuid.Must(uuid.NewV7()).String(), bookID, bookPath); err != nil {
		t.Fatalf("seed book file: %v", err)
	}

	server := NewHTTPServer()
	server.SetupServer(db, cache.NewRamCache())
	app := server.App

	signin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin",
		strings.NewReader(`{"email":"reader@example.com","password":"Sup3rSecret!"}`))
	signin.Header.Set("Content-Type", "application/json")
	signinResp, err := app.Test(signin)
	if err != nil {
		t.Fatalf("signin failed: %v", err)
	}
	var auth struct {
		Status bool `json:"status"`
		Data   struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	signinBody, _ := io.ReadAll(signinResp.Body)
	if err := json.Unmarshal(signinBody, &auth); err != nil || !auth.Status {
		t.Fatalf("signin bad response: %v (%s)", err, signinBody)
	}

	send := httptest.NewRequest(http.MethodPost, "/api/v1/books/"+bookID+"/send-email",
		strings.NewReader(`{"recipient_email":"kindle@example.com"}`))
	send.Header.Set("Content-Type", "application/json")
	send.Header.Set("Authorization", "Bearer "+auth.Data.AccessToken)
	sendResp, err := app.Test(send, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("send request failed: %v", err)
	}
	sendBody, _ := io.ReadAll(sendResp.Body)
	if sendResp.StatusCode != http.StatusOK {
		t.Fatalf("send-email returned %d: %s", sendResp.StatusCode, sendBody)
	}

	select {
	case message := <-received:
		for _, want := range []string{"library@example.com", "kindle@example.com", "Dune"} {
			if !strings.Contains(message, want) {
				t.Errorf("delivered message is missing %q:\n%s", want, message)
			}
		}
	case <-time.After(20 * time.Second):
		t.Fatal("configured SMTP server never received the message")
	}
}

func TestSMTPTestEndpointRequiresPermission(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	t.Setenv("DB_ENCRYPTION_KEY", "smtp-e2e-test-key")
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/smtp/test",
		strings.NewReader(`{"host":"smtp.example.com","port":587}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("anonymous SMTP test returned %d, want 401/403: %s", resp.StatusCode, body)
	}
}
