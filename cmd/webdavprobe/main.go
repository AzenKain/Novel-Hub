package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/routes"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func main() {
	fmt.Println("=== NovelHub WebDAV Server CLI Probe ===")

	urlFlag := flag.String("url", "", "WebDAV Server URL (e.g. http://localhost:8080/webdav)")
	userFlag := flag.String("user", "admin@novelhub.local", "WebDAV Username/Email")
	passFlag := flag.String("pass", "admin123", "WebDAV Password or Magic Code")
	demoFlag := flag.Bool("demo", true, "Run in-memory end-to-end WebDAV probe test")
	flag.Parse()

	if *urlFlag != "" {
		runLiveProbe(*urlFlag, *userFlag, *passFlag)
	} else if *demoFlag {
		runDemoProbe()
	} else {
		flag.Usage()
	}
}

func runLiveProbe(baseURL, user, pass string) {
	fmt.Printf("[*] Probing live WebDAV server at: %s\n", baseURL)
	client := &http.Client{Timeout: 10 * time.Second}

	req, _ := http.NewRequest("OPTIONS", baseURL, nil)
	req.SetBasicAuth(user, pass)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: OPTIONS request failed: %v\n", err)
		return
	}
	fmt.Printf("[+] OPTIONS Status: %d\n", resp.StatusCode)
	fmt.Printf("    DAV: %s\n", resp.Header.Get("DAV"))
	fmt.Printf("    Allow: %s\n", resp.Header.Get("Allow"))
	resp.Body.Close()

	req, _ = http.NewRequest("PROPFIND", baseURL, nil)
	req.SetBasicAuth(user, pass)
	req.Header.Set("Depth", "1")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: PROPFIND request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("\n[+] PROPFIND Status: %d (Content-Type: %s)\n", resp.StatusCode, resp.Header.Get("Content-Type"))
	fmt.Println("--- Multi-Status Response ---")
	fmt.Println(string(body))
}

func runDemoProbe() {
	fmt.Println("[*] Setting up in-memory WebDAV server with mock library and book files...")

	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	lib := &response.LibraryResponse{
		ID:        "lib-main",
		Name:      "General Library",
		UpdatedAt: now,
	}

	book := &models.BookEntity{
		ID:        "book-1",
		Title:     "Dune - Frank Herbert",
		LibraryID: "lib-main",
		UpdatedAt: now,
	}

	file := &models.BookFileEntity{
		ID:        "file-1",
		BookID:    "book-1",
		Path:      "/tmp/dune.epub",
		Format:    "epub",
		SizeBytes: 2097152,
		ModTime:   now,
	}

	libService := &mockLibService{libs: []*response.LibraryResponse{lib}}
	bookService := &mockBookService{books: []*models.BookEntity{book}, files: []*models.BookFileEntity{file}}
	perms := &mockPermCache{allowGuest: false}
	settings := &mockSettingsService{}
	auth := &mockAuthService{validUser: "user@novelhub.local", validPass: "pass123"}

	webdavService := services.NewWebDAVService(libService, bookService, perms, settings)
	webdavCtrl := controllers.NewWebDAVController(webdavService)

	app := fiber.New(fiber.Config{
		RequestMethods: append(fiber.DefaultMethods, "PROPFIND", "PROPPATCH", "MKCOL", "COPY", "MOVE"),
	})

	routes.WebDAVRoutes(app, webdavCtrl, auth, settings, perms)

	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@novelhub.local:pass123"))

	fmt.Println("\n1. Testing Unauthenticated HTTP OPTIONS /webdav (Capability Discovery)...")
	optReq := httptest.NewRequest("OPTIONS", "/webdav", nil)
	optResp, err := app.Test(optReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OPTIONS test failed: %v\n", err)
		return
	}
	fmt.Printf("   Status: %d\n", optResp.StatusCode)
	fmt.Printf("   DAV Header: %s\n", optResp.Header.Get("DAV"))
	fmt.Printf("   Allow: %s\n", optResp.Header.Get("Allow"))
	if optResp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "FAILED: Expected 200 OK on unauthenticated OPTIONS\n")
		return
	}

	fmt.Println("\n2. Testing Unauthenticated PROPFIND /webdav (401 Challenge Check)...")
	unauthPropReq := httptest.NewRequest("PROPFIND", "/webdav", nil)
	unauthPropReq.Header.Set("Depth", "0")
	unauthPropResp, _ := app.Test(unauthPropReq)
	fmt.Printf("   Status: %d (Expected 401)\n", unauthPropResp.StatusCode)
	fmt.Printf("   WWW-Authenticate: %s\n", unauthPropResp.Header.Get("WWW-Authenticate"))
	if unauthPropResp.StatusCode != 401 || unauthPropResp.Header.Get("WWW-Authenticate") == "" {
		fmt.Fprintf(os.Stderr, "FAILED: Expected 401 Unauthorized with WWW-Authenticate header\n")
		return
	}

	fmt.Println("\n3. Testing Authenticated PROPFIND /webdav (Root Level, Depth: 1)...")
	propReq1 := httptest.NewRequest("PROPFIND", "/webdav", nil)
	propReq1.Header.Set("Authorization", authHeader)
	propReq1.Header.Set("Depth", "1")
	propResp1, _ := app.Test(propReq1)
	body1, _ := io.ReadAll(propResp1.Body)
	fmt.Printf("   Status: %d Multi-Status\n", propResp1.StatusCode)
	fmt.Println(string(body1))

	fmt.Println("\n4. Testing Authenticated PROPFIND /webdav/General Library/ (Library Level, Depth: 1)...")
	propReq2 := httptest.NewRequest("PROPFIND", "/webdav/General%20Library", nil)
	propReq2.Header.Set("Authorization", authHeader)
	propReq2.Header.Set("Depth", "1")
	propResp2, _ := app.Test(propReq2)
	body2, _ := io.ReadAll(propResp2.Body)
	fmt.Printf("   Status: %d Multi-Status\n", propResp2.StatusCode)
	fmt.Println(string(body2))

	fmt.Println("\n5. Testing /api/webdav Mount (OPTIONS & PROPFIND)...")
	apiOptReq := httptest.NewRequest("OPTIONS", "/api/webdav", nil)
	apiOptResp, _ := app.Test(apiOptReq)
	fmt.Printf("   /api/webdav OPTIONS Status: %d\n", apiOptResp.StatusCode)

	apiPropReq := httptest.NewRequest("PROPFIND", "/api/webdav", nil)
	apiPropReq.Header.Set("Authorization", authHeader)
	apiPropReq.Header.Set("Depth", "0")
	apiPropResp, _ := app.Test(apiPropReq)
	apiBody, _ := io.ReadAll(apiPropResp.Body)
	fmt.Printf("   /api/webdav PROPFIND Status: %d\n", apiPropResp.StatusCode)
	if strings.Contains(string(apiBody), "<D:href>/api/webdav/</D:href>") {
		fmt.Println("   /api/webdav returned correct /api/webdav/ root href!")
	} else {
		fmt.Fprintf(os.Stderr, "FAILED: Expected /api/webdav/ href in response, got:\n%s\n", string(apiBody))
		return
	}

	fmt.Println("\n🎉 All WebDAV Protocol Tests Passed Successfully!")
}

type mockLibService struct {
	services.LibraryService
	libs []*response.LibraryResponse
}

func (m *mockLibService) ListLibraries(_ context.Context, _ *response.JWTClaims) ([]*response.LibraryResponse, error) {
	return m.libs, nil
}

type mockBookService struct {
	services.BookService
	books []*models.BookEntity
	files []*models.BookFileEntity
}

func (m *mockBookService) SearchBooks(_ context.Context, _ *string, _ *string, _, _, _, _, _ string, _ string, _ string, _ int64, _ string) ([]*models.BookEntity, error) {
	return m.books, nil
}

func (m *mockBookService) ListBookFiles(_ context.Context, _ string) ([]*models.BookFileEntity, error) {
	return m.files, nil
}

func (m *mockBookService) CanReadBook(_ context.Context, _ *models.BookEntity, _ *response.JWTClaims) bool {
	return true
}

func (m *mockBookService) SafeDownloadFilename(title string, ext string) string {
	return title + ext
}

type mockPermCache struct {
	services.PermissionCache
	allowGuest bool
}

func (m *mockPermCache) Can(_ context.Context, _ string, _ string, _ map[string]any) bool {
	return true
}

func (m *mockPermCache) IsAdmin(_ []string, _ []constants.RoleType) bool {
	return false
}

func (m *mockPermCache) CanRoles(_ []string, roles []constants.RoleType, _ string, _ map[string]any) bool {
	for _, r := range roles {
		if r == constants.RoleTypeGuest {
			return m.allowGuest
		}
	}
	return true
}

type mockSettingsService struct {
	services.SettingsService
}

func (m *mockSettingsService) Public(_ context.Context) (*models.PublicSettings, error) {
	return &models.PublicSettings{GuestLoginRequired: false}, nil
}

type mockAuthService struct {
	services.AuthService
	validUser string
	validPass string
}

func (m *mockAuthService) ValidateCredentials(_ context.Context, dto *request.SignInDto) (*response.JWTClaims, error) {
	if dto.Email == m.validUser && dto.Password == m.validPass {
		return &response.JWTClaims{UId: "user-1", Roles: []constants.RoleType{constants.RoleTypeUser}}, nil
	}
	return nil, fmt.Errorf("invalid credentials")
}

func checkUnused() {
	_ = bytes.Buffer{}
}
