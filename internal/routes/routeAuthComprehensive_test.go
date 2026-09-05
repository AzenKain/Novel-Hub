package routes_test

import (
	"bytes"
	"context"
	"database/sql"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"novelhub/internal/controllers"
	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/internal/routes"
	"novelhub/internal/services"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
)

type testHarness struct {
	app             *fiber.App
	db              *sql.DB
	ramCache        cache.Cache
	userRepo        repositories.UserRepository
	roleRepo        repositories.RoleRepository
	bookRepo        repositories.BookDBRepository
	libraryRepo     repositories.LibraryRepository
	featureRepo     repositories.FeatureRepository
	permissionCache services.PermissionCache
	uploadService   services.UploadService
	customization   services.CustomizationService
	secret          string
}

func setupTestHarness(t *testing.T) *testHarness {
	t.Helper()
	secret := "test-secret-key-1234567890123456"
	os.Setenv("JWT_SECRET", secret)
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-key-1234567890123456")

	dataDir := filepath.Join(t.TempDir(), "data")
	_ = os.MkdirAll(filepath.Join(dataDir, "uploads"), 0755)
	_ = os.MkdirAll(filepath.Join(dataDir, "soundscapes"), 0755)
	_ = os.MkdirAll(filepath.Join(dataDir, "fonts"), 0755)
	os.Setenv("DATA_DIR", dataDir)

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	ramCache := cache.NewRamCache()
	userRepo := repositories.NewUserRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	libraryRepo := repositories.NewLibraryRepository(db, ramCache)
	featureRepo := repositories.NewFeatureRepository(db, ramCache)
	customizationRepo := repositories.NewCustomizationRepository(db, ramCache)

	permissionCache := services.NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}

	parserReg := bookparser.NewRegistry()
	txManager := database.NewTxManager(db)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	settingsService := services.NewSettingsService(settingsRepo, txManager, permissionCache)
	_ = settingsService.Reload(context.Background())

	bookFileRepo, _ := repositories.NewBookFileRepository(filepath.Join(dataDir, "books"))
	bookService := services.NewBookService(bookRepo, featureRepo, libraryRepo, bookFileRepo, parserReg, txManager, settingsService, permissionCache, nil, nil)
	libraryService := services.NewLibraryService(libraryRepo, bookRepo, bookFileRepo, parserReg, permissionCache, settingsService, nil)
	uploadService := services.NewUploadService(libraryService, bookService, libraryRepo, permissionCache, settingsService)
	customizationService := services.NewCustomizationService(customizationRepo, permissionCache, settingsService, dataDir)

	app := fiber.New()
	v1 := app.Group("/api/v1")

	uploadController := controllers.NewUploadController(uploadService)
	customizationController := controllers.NewCustomizationController(customizationService, settingsService)
	routes.SetupUploadRoutes(v1, uploadController, userRepo, permissionCache)
	routes.CustomizationRoutes(v1, customizationController, userRepo, permissionCache)

	return &testHarness{
		app:             app,
		db:              db,
		ramCache:        ramCache,
		userRepo:        userRepo,
		roleRepo:        roleRepo,
		bookRepo:        bookRepo,
		libraryRepo:     libraryRepo,
		featureRepo:     featureRepo,
		permissionCache: permissionCache,
		uploadService:   uploadService,
		customization:   customizationService,
		secret:          secret,
	}
}

func generateTestToken(secret, uid string, roles []constants.RoleType, roleIDs []string, tokenVersion int32) string {
	claims := &response.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "novelhub",
			Subject:   uid,
			Audience:  jwt.ClaimStrings{"novelhub-access"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		UId:          uid,
		Roles:        roles,
		RoleIDs:      roleIDs,
		TokenType:    "access",
		TokenVersion: tokenVersion,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

func TestUploadRouteAuthorizationAndFormatValidation(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	libID := uuid.Must(uuid.NewV7()).String()
	_, err := h.db.Exec(`INSERT INTO libraries (id, name) VALUES (?, 'Main Library')`, libID)
	if err != nil {
		t.Fatal(err)
	}

	readerID := uuid.Must(uuid.NewV7()).String()
	_, err = h.db.Exec(`INSERT INTO users (id, email, password_hash, token_version) VALUES (?, 'read_only@example.com', 'hash', 1)`, readerID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO roles (id, name) VALUES ('role-reader', 'READER')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, 'role-reader')`, readerID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO role_permissions (id, role_id, permission_key, effect) VALUES ('p1', 'role-reader', 'book.read', 'allow')`)
	if err != nil {
		t.Fatal(err)
	}
	_ = h.permissionCache.Reload(ctx)

	uploaderID := uuid.Must(uuid.NewV7()).String()
	_, err = h.db.Exec(`INSERT INTO users (id, email, password_hash, token_version) VALUES (?, 'uploader@example.com', 'hash', 1)`, uploaderID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO roles (id, name) VALUES ('role-uploader', 'UPLOADER')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, 'role-uploader')`, uploaderID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO role_permissions (id, role_id, permission_key, effect) VALUES ('p2', 'role-uploader', 'book.upload', 'allow')`)
	if err != nil {
		t.Fatal(err)
	}
	_ = h.ramCache.DelByPattern(ctx, "*")
	_ = h.permissionCache.Reload(ctx)

	readerToken := generateTestToken(h.secret, readerID, []constants.RoleType{}, []string{"role-reader"}, 1)
	uploaderToken := generateTestToken(h.secret, uploaderID, []constants.RoleType{}, []string{"role-uploader"}, 1)

	t.Run("No token on upload init returns 401", func(t *testing.T) {
		body, _ := jsonx.Marshal(request.InitUploadDto{
			Target:      "library",
			LibraryID:   libID,
			Filename:    "test.epub",
			TotalBytes:  1024,
			TotalChunks: 1,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/init", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	t.Run("User without PermBookUpload returns 403", func(t *testing.T) {
		body, _ := jsonx.Marshal(request.InitUploadDto{
			Target:      "library",
			LibraryID:   libID,
			Filename:    "test.epub",
			TotalBytes:  1024,
			TotalChunks: 1,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/init", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+readerToken)
		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected status 403 Forbidden, got %d", resp.StatusCode)
		}
	})

	t.Run("Attempt to upload unsupported file format (e.g. .exe / .sh) returns 400", func(t *testing.T) {
		for _, badFile := range []string{"malware.exe", "script.sh", "payload.bin", "trojan.dll", "hack.php"} {
			body, _ := jsonx.Marshal(request.InitUploadDto{
				Target:      "library",
				LibraryID:   libID,
				Filename:    badFile,
				TotalBytes:  1024,
				TotalChunks: 1,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/init", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+uploaderToken)
			resp, err := h.app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for %s, got %d", badFile, resp.StatusCode)
			}
		}
	})

	t.Run("Valid book format upload init succeeds with 200", func(t *testing.T) {
		body, _ := jsonx.Marshal(request.InitUploadDto{
			Target:      "library",
			LibraryID:   libID,
			Filename:    "great_novel.epub",
			TotalBytes:  1024,
			TotalChunks: 1,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/init", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+uploaderToken)
		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
		}
	})
}

func TestCustomizationRoutesPermissions(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	soundUserID := uuid.Must(uuid.NewV7()).String()
	_, err := h.db.Exec(`INSERT INTO users (id, email, password_hash, token_version) VALUES (?, 'sound_user@example.com', 'hash', 1)`, soundUserID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO roles (id, name) VALUES ('role-sound', 'SOUND_ROLE')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, 'role-sound')`, soundUserID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO role_permissions (id, role_id, permission_key, effect) VALUES ('p3', 'role-sound', 'user.soundscape.manage', 'allow')`)
	if err != nil {
		t.Fatal(err)
	}

	plainUserID := uuid.Must(uuid.NewV7()).String()
	_, err = h.db.Exec(`INSERT INTO users (id, email, password_hash, token_version) VALUES (?, 'plain_user@example.com', 'hash', 1)`, plainUserID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO roles (id, name) VALUES ('role-restricted', 'RESTRICTED_ROLE')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, 'role-restricted')`, plainUserID)
	if err != nil {
		t.Fatal(err)
	}

	_ = h.ramCache.DelByPattern(ctx, "*")
	_ = h.permissionCache.Reload(ctx)

	soundToken := generateTestToken(h.secret, soundUserID, []constants.RoleType{}, []string{"role-sound"}, 1)
	plainToken := generateTestToken(h.secret, plainUserID, []constants.RoleType{}, []string{"role-restricted"}, 1)

	adminUserID := uuid.Must(uuid.NewV7()).String()
	_, _ = h.db.Exec(`INSERT INTO users (id, email, password_hash, token_version) VALUES (?, 'admin_cust@example.com', 'hash', 1)`, adminUserID)
	adminToken := generateTestToken(h.secret, adminUserID, []constants.RoleType{constants.RoleTypeAdmin}, []string{"admin"}, 1)

	t.Run("User without soundscape permission is rejected with 403 on soundscape upload", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("title", "Rain")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/soundscapes/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+plainToken)

		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
		}
	})

	t.Run("User without font permission is rejected with 403 on font upload", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("name", "CustomFont")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fonts/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+soundToken)

		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
		}
	})

	t.Run("Private soundscape is accessible to owner but forbidden to unauthenticated guest", func(t *testing.T) {
		soundFile := filepath.Join(os.Getenv("DATA_DIR"), "soundscapes", "test_private.mp3")
		_ = os.WriteFile(soundFile, []byte("ID3_DUMMY_AUDIO_DATA"), 0644)

		soundID := uuid.Must(uuid.NewV7()).String()
		_, err := h.db.Exec(`INSERT INTO soundscapes (id, user_id, name, file_path, is_system) VALUES (?, ?, 'Private Rain', ?, 0)`, soundID, soundUserID, soundFile)
		if err != nil {
			t.Fatal(err)
		}
		_ = h.ramCache.DelByPattern(ctx, "*")

		reqGuest := httptest.NewRequest(http.MethodGet, "/api/v1/soundscapes/"+soundID+"/stream", nil)
		respGuest, err := h.app.Test(reqGuest)
		if err != nil {
			t.Fatal(err)
		}
		if respGuest.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for guest accessing private soundscape, got %d", respGuest.StatusCode)
		}

		reqOther := httptest.NewRequest(http.MethodGet, "/api/v1/soundscapes/"+soundID+"/stream", nil)
		reqOther.Header.Set("Authorization", "Bearer "+plainToken)
		respOther, err := h.app.Test(reqOther)
		if err != nil {
			t.Fatal(err)
		}
		if respOther.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for other user accessing private soundscape, got %d", respOther.StatusCode)
		}

		reqOwner := httptest.NewRequest(http.MethodGet, "/api/v1/soundscapes/"+soundID+"/stream", nil)
		reqOwner.Header.Set("Authorization", "Bearer "+soundToken)
		respOwner, err := h.app.Test(reqOwner)
		if err != nil {
			t.Fatal(err)
		}
		if respOwner.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for owner accessing private soundscape, got %d", respOwner.StatusCode)
		}
	})

	t.Run("System soundscape is accessible to everyone including guests without auth", func(t *testing.T) {
		sysSoundFile := filepath.Join(os.Getenv("DATA_DIR"), "soundscapes", "test_system.mp3")
		_ = os.WriteFile(sysSoundFile, []byte("ID3_SYSTEM_AUDIO_DATA"), 0644)

		sysSoundID := uuid.Must(uuid.NewV7()).String()
		_, err := h.db.Exec(`INSERT INTO soundscapes (id, user_id, name, file_path, is_system) VALUES (?, NULL, 'System Coffee Shop', ?, 1)`, sysSoundID, sysSoundFile)
		if err != nil {
			t.Fatal(err)
		}
		_ = h.ramCache.DelByPattern(ctx, "*")

		reqGuest := httptest.NewRequest(http.MethodGet, "/api/v1/soundscapes/"+sysSoundID+"/stream", nil)
		respGuest, err := h.app.Test(reqGuest)
		if err != nil {
			t.Fatal(err)
		}
		if respGuest.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for guest accessing system soundscape, got %d", respGuest.StatusCode)
		}
	})

	t.Run("Soundscape upload rejects non-audio file or fake magic header with 400", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("name", "Malicious Sound")
		part, _ := writer.CreateFormFile("audio", "malware.exe")
		_, _ = part.Write([]byte("MZ_WINDOWS_BINARY_HEADER"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/soundscapes/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+soundToken)

		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for fake audio file, got %d", resp.StatusCode)
		}
	})

	t.Run("Soundscape upload succeeds with valid ID3 MP3 audio file", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("name", "Valid Birds Chirping")
		_ = writer.WriteField("category", "nature")
		part, _ := writer.CreateFormFile("audio", "birds.mp3")
		_, _ = part.Write([]byte("ID3_VALID_MP3_AUDIO_HEADER_AND_CONTENT"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/soundscapes/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+soundToken)

		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 Created for valid soundscape, got %d", resp.StatusCode)
		}
	})

	t.Run("Font upload rejects non-font file with 400 Bad Request", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("name", "Fake Font")
		_ = writer.WriteField("font_family", "FakeSerif")
		_ = writer.WriteField("source_type", "file")
		part, _ := writer.CreateFormFile("font", "font.woff2")
		_, _ = part.Write([]byte("ELF_NOT_A_FONT"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fonts/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for fake font header, got %d", resp.StatusCode)
		}
	})

	t.Run("Font upload succeeds with valid wOF2 font header", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("name", "Merriweather Woff2")
		_ = writer.WriteField("font_family", "Merriweather")
		_ = writer.WriteField("source_type", "file")
		part, _ := writer.CreateFormFile("font", "merriweather.woff2")
		_, _ = part.Write([]byte("wOF2_VALID_FONT_DATA"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fonts/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 Created for valid font, got %d", resp.StatusCode)
		}
	})

	t.Run("Theme creation rejects XSS injection in Custom CSS with 400 Bad Request", func(t *testing.T) {
		themePayload := `{"name":"XSS Theme","bg_color":"#fff","text_color":"#000","accent_color":"#f00","custom_css":"</style><script>alert(document.cookie)</script>"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/themes", bytes.NewBufferString(themePayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for XSS in custom CSS, got %d", resp.StatusCode)
		}
	})

	t.Run("Theme creation succeeds with safe Custom CSS", func(t *testing.T) {
		themePayload := `{"name":"Nord Dark","bg_color":"#2e3440","text_color":"#d8dee9","accent_color":"#88c0d0","custom_css":".reader-container { line-height: 1.8; }"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/themes", bytes.NewBufferString(themePayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := h.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 Created for safe theme, got %d", resp.StatusCode)
		}
	})
}
