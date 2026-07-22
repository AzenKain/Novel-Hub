package services

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
)

var (
	availableSidebarItems = []string{
		"books",
		"hot_books",
		"downloaded_books",
		"top_rated_books",
		"bookmarked_books",
		"read_books",
		"unread_books",
		"subjects",
		"series",
		"authors",
		"publishers",
		"languages",
		"file_formats",
		"ratings",
		"archived_books",
		"collections",
	}
	availableHomeSections = []string{"random_books", "top_books"}
	availableGuestModes   = []string{"all", "selected_libraries", "login_required"}
)

type SettingsService interface {
	Reload(ctx context.Context) error
	Public(ctx context.Context) (*models.PublicSettings, error)
	UpdateSettings(ctx context.Context, settings map[string]any) (*models.PublicSettings, error)
	GuestAllows(libraryID string) bool
	SetupRequired(ctx context.Context) bool
	SaveAsset(ctx context.Context, target string, fileData []byte, fileName string, urlStr string) (string, error)
}

type settingsService struct {
	repo        repositories.SettingsRepository
	permissions PermissionCache
	mu          sync.RWMutex
	data        *models.PublicSettings
	raw         map[string]any
}

func NewSettingsService(repo repositories.SettingsRepository, permissions ...PermissionCache) SettingsService {
	var permCache PermissionCache
	if len(permissions) > 0 {
		permCache = permissions[0]
	}
	return &settingsService{
		repo:        repo,
		permissions: permCache,
		raw:         map[string]any{},
		data:        defaultPublicSettings(),
	}
}

func defaultPublicSettings() *models.PublicSettings {
	return &models.PublicSettings{
		Site: models.SiteSettings{
			Title:           "NovelHub",
			Description:     "Local novel library manager",
			MetaDescription: "Self-hosted local-first digital book library manager.",
		},
		SidebarVisibleItems:   append([]string(nil), availableSidebarItems...),
		HomeSections:          models.HomeSectionSettings{RandomBooks: true, TopBooks: true},
		RegistrationEnabled:   true,
		GuestAccess:           models.LibraryPolicy{Mode: "all", LibraryIDs: []string{}},
		GuestPermissions:      constants.GetDefaultPermissionsForRole(constants.RoleTypeGuest),
		SetupCompleted:        true,
		AvailableSidebarItems: append([]string(nil), availableSidebarItems...),
		AvailableHomeSections: append([]string(nil), availableHomeSections...),
		AvailableGuestModes:   append([]string(nil), availableGuestModes...),
	}
}

func (s *settingsService) Reload(ctx context.Context) error {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	raw := make(map[string]any, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		var value any
		if err := jsonx.Unmarshal([]byte(row.ValueJSON), &value); err != nil {
			continue
		}
		raw[row.Key] = value
	}
	settings := settingsFromRaw(raw)
	settings.SetupCompleted = s.setupCompleted(ctx)

	s.mu.Lock()
	s.raw = raw
	s.data = settings
	s.mu.Unlock()
	return nil
}

func (s *settingsService) Public(ctx context.Context) (*models.PublicSettings, error) {
	s.mu.RLock()
	current := s.data
	s.mu.RUnlock()
	if current == nil {
		if err := s.Reload(ctx); err != nil {
			return nil, err
		}
		s.mu.RLock()
		current = s.data
		s.mu.RUnlock()
	}
	copyValue := *current
	copyValue.SidebarVisibleItems = append([]string(nil), current.SidebarVisibleItems...)
	copyValue.GuestAccess.LibraryIDs = append([]string(nil), current.GuestAccess.LibraryIDs...)
	if s.permissions != nil {
		if dynamicGuestPerms := s.permissions.GetGuestPermissions(); len(dynamicGuestPerms) > 0 {
			copyValue.GuestPermissions = dynamicGuestPerms
		} else {
			copyValue.GuestPermissions = append([]string(nil), current.GuestPermissions...)
		}
	} else {
		copyValue.GuestPermissions = append([]string(nil), current.GuestPermissions...)
	}
	return &copyValue, nil
}

func (s *settingsService) UpdateSettings(ctx context.Context, settings map[string]any) (*models.PublicSettings, error) {
	for key, value := range settings {
		if !allowedSettingKey(key) {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Unsupported setting: "+key)
		}
		data, err := jsonx.Marshal(value)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid setting value")
		}
		if err := s.repo.Upsert(ctx, key, string(data)); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save settings")
		}
	}
	if err := s.Reload(ctx); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to reload settings")
	}
	public, err := s.Public(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load settings")
	}
	return public, nil
}

func (s *settingsService) GuestAllows(libraryID string) bool {
	s.mu.RLock()
	current := s.data
	s.mu.RUnlock()
	if current == nil {
		current = defaultPublicSettings()
	}
	switch current.GuestAccess.Mode {
	case "login_required":
		return false
	case "selected_libraries":
		return libraryID != "" && slices.Contains(current.GuestAccess.LibraryIDs, libraryID)
	default:
		return true
	}
}

func (s *settingsService) SetupRequired(ctx context.Context) bool {
	settings, err := s.Public(ctx)
	if err != nil {
		return false
	}
	if !settings.SetupCompleted {
		return true
	}
	count, err := s.repo.CountAdminUsers(ctx)
	if err != nil {
		return false
	}
	return count == 0
}

func (s *settingsService) setupCompleted(ctx context.Context) bool {
	value, err := s.repo.GetSetupState(ctx, "completed")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		return true
	}
	if value != "true" {
		return false
	}
	count, err := s.repo.CountAdminUsers(ctx)
	if err != nil {
		return true
	}
	return count > 0
}

func settingsFromRaw(raw map[string]any) *models.PublicSettings {
	settings := defaultPublicSettings()
	settings.Site.Title = rawString(raw, "site.title", settings.Site.Title)
	settings.Site.Description = rawString(raw, "site.description", settings.Site.Description)
	settings.Site.Favicon = rawString(raw, "site.favicon", settings.Site.Favicon)
	settings.Site.Logo = rawString(raw, "site.logo", settings.Site.Logo)
	settings.Site.MetaDescription = rawString(raw, "site.meta_description", settings.Site.MetaDescription)
	settings.SidebarVisibleItems = filterKnown(rawStringSlice(raw, "sidebar.visible_items", settings.SidebarVisibleItems), availableSidebarItems)
	settings.HomeSections = rawHomeSections(raw, settings.HomeSections)
	settings.RegistrationEnabled = rawBool(raw, "auth.registration_enabled", settings.RegistrationEnabled)
	settings.GuestAccess = rawPolicy(raw, "guest_access", settings.GuestAccess, availableGuestModes)
	settings.EnableInBookSearch = rawBool(raw, "reader.enable_in_book_search", false)
	settings.EnableCustomFontUpload = rawBool(raw, "font.enable_custom_font_upload", false)
	return settings
}

func rawString(raw map[string]any, key string, fallback string) string {
	value, ok := raw[key]
	if !ok {
		return fallback
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return fallback
}

func rawBool(raw map[string]any, key string, fallback bool) bool {
	value, ok := raw[key]
	if !ok {
		return fallback
	}
	if typed, ok := value.(bool); ok {
		return typed
	}
	return fallback
}

func rawStringSlice(raw map[string]any, key string, fallback []string) []string {
	value, ok := raw[key]
	if !ok {
		return append([]string(nil), fallback...)
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return append([]string(nil), fallback...)
	}
}

func rawHomeSections(raw map[string]any, fallback models.HomeSectionSettings) models.HomeSectionSettings {
	value, ok := raw["home.sections"].(map[string]any)
	if !ok {
		return fallback
	}
	return models.HomeSectionSettings{
		RandomBooks: mapBool(value, "random_books", fallback.RandomBooks),
		TopBooks:    mapBool(value, "top_books", fallback.TopBooks),
	}
}

func rawPolicy(raw map[string]any, prefix string, fallback models.LibraryPolicy, allowedModes []string) models.LibraryPolicy {
	mode := rawString(raw, prefix+".mode", fallback.Mode)
	if !slices.Contains(allowedModes, mode) {
		mode = fallback.Mode
	}
	return models.LibraryPolicy{
		Mode:       mode,
		LibraryIDs: rawStringSlice(raw, prefix+".library_ids", fallback.LibraryIDs),
	}
}

func mapBool(raw map[string]any, key string, fallback bool) bool {
	value, ok := raw[key]
	if !ok {
		return fallback
	}
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func filterKnown(items []string, known []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if slices.Contains(known, item) && !slices.Contains(out, item) {
			out = append(out, item)
		}
	}
	return out
}

func allowedSettingKey(key string) bool {
	switch key {
	case "site.title",
		"site.description",
		"site.favicon",
		"site.logo",
		"site.meta_description",
		"sidebar.visible_items",
		"home.sections",
		"auth.registration_enabled",
		"guest_access.mode",
		"guest_access.library_ids",
		"reader.enable_in_book_search",
		"font.enable_custom_font_upload":
		return true
	default:
		return false
	}
}

func (s *settingsService) SaveAsset(ctx context.Context, target string, fileData []byte, fileName string, urlStr string) (string, error) {
	publicDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to create public directory")
	}

	outFilename := "logo.png"
	if target == "favicon" {
		outFilename = "favicon.png"
	}
	destPath := filepath.Join(publicDir, outFilename)

	if len(fileData) > 0 {
		if err := os.WriteFile(destPath, fileData, 0644); err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to save file")
		}
		return "/public/" + outFilename, nil
	}

	if urlStr != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if err != nil {
			return "", apperrors.New(apperrors.ErrBadRequest, "Invalid URL")
		}
		client := netx.NewSafeHTTPClient(15 * time.Second)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			return "", apperrors.New(apperrors.ErrBadRequest, "Failed to fetch URL or URL blocked for security")
		}
		defer resp.Body.Close()

		out, err := os.Create(destPath)
		if err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to create destination file")
		}
		defer out.Close()

		if _, err := io.Copy(out, io.LimitReader(resp.Body, 10<<20)); err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to write downloaded asset")
		}
		return "/public/" + outFilename, nil
	}

	return "", apperrors.New(apperrors.ErrBadRequest, "Provide a file or URL")
}
