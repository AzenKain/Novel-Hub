package services

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
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
	Admin(ctx context.Context) (*models.AdminSettings, error)
	Limits() models.RuntimeLimits
	UpdateSettings(ctx context.Context, settings map[string]any) (*models.AdminSettings, error)
	GuestAllows(libraryID string) bool
	SetupRequired(ctx context.Context) bool
	SaveAsset(ctx context.Context, target string, fileData []byte, fileName string, urlStr string) (string, error)
}

type settingsService struct {
	repo        repositories.SettingsRepository
	permissions PermissionCache
	txManager   database.TxManager
	updateMu    sync.Mutex
	mu          sync.RWMutex
	data        *models.PublicSettings
	limits      models.RuntimeLimits
	raw         map[string]any
}

func NewSettingsService(repo repositories.SettingsRepository, txManager database.TxManager, permissions ...PermissionCache) SettingsService {
	var permCache PermissionCache
	if len(permissions) > 0 {
		permCache = permissions[0]
	}
	return &settingsService{
		repo:        repo,
		permissions: permCache,
		txManager:   txManager,
		raw:         map[string]any{},
		data:        defaultPublicSettings(),
		limits:      defaultRuntimeLimits(),
	}
}

func defaultRuntimeLimits() models.RuntimeLimits {
	return models.RuntimeLimits{
		UploadChunkBytes:        constants.MaxUploadChunkBytes,
		UploadChunks:            constants.MaxUploadChunks,
		UploadSessions:          constants.MaxUploadSessions,
		UploadBytes:             constants.MaxUploadBytes,
		UploadSessionTTLSeconds: int64(constants.UploadSessionTTL / time.Second),
		CoverBytes:              constants.MaxCoverBytes,
		SiteAssetBytes:          constants.MaxSiteAssetBytes,
	}
}

func runtimeLimitBounds() models.RuntimeLimitBounds {
	return models.RuntimeLimitBounds{
		Min: models.RuntimeLimits{
			UploadChunkBytes:        constants.MinRuntimeUploadChunkBytes,
			UploadChunks:            constants.MinRuntimeUploadChunks,
			UploadSessions:          constants.MinRuntimeUploadSessions,
			UploadBytes:             constants.MinRuntimeUploadBytes,
			UploadSessionTTLSeconds: int64(constants.MinRuntimeUploadSessionTTL / time.Second),
			CoverBytes:              constants.MinRuntimeCoverBytes,
			SiteAssetBytes:          constants.MinRuntimeSiteAssetBytes,
		},
		Max: models.RuntimeLimits{
			UploadChunkBytes:        constants.HardMaxUploadChunkBytes,
			UploadChunks:            constants.HardMaxUploadChunks,
			UploadSessions:          constants.HardMaxUploadSessions,
			UploadBytes:             constants.HardMaxUploadBytes,
			UploadSessionTTLSeconds: int64(constants.HardMaxUploadSessionTTL / time.Second),
			CoverBytes:              constants.HardMaxCoverBytes,
			SiteAssetBytes:          constants.HardMaxSiteAssetBytes,
		},
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
			if strings.HasPrefix(row.Key, "limits.") {
				return errors.New("Invalid runtime limit: " + row.Key)
			}
			continue
		}
		raw[row.Key] = value
	}
	settings := settingsFromRaw(raw)
	settings.SetupCompleted = s.setupCompleted(ctx)
	limits, err := runtimeLimitsFromRaw(raw)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.raw = raw
	s.data = settings
	s.limits = limits
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

func (s *settingsService) Admin(ctx context.Context) (*models.AdminSettings, error) {
	public, err := s.Public(ctx)
	if err != nil {
		return nil, err
	}
	return &models.AdminSettings{
		PublicSettings: *public,
		Limits:         s.Limits(),
		Bounds:         runtimeLimitBounds(),
	}, nil
}

func (s *settingsService) Limits() models.RuntimeLimits {
	s.mu.RLock()
	limits := s.limits
	s.mu.RUnlock()
	return limits
}

func (s *settingsService) UpdateSettings(ctx context.Context, settings map[string]any) (*models.AdminSettings, error) {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	s.mu.RLock()
	candidateRaw := make(map[string]any, len(s.raw)+len(settings))
	for key, value := range s.raw {
		candidateRaw[key] = value
	}
	s.mu.RUnlock()

	keys := make([]string, 0, len(settings))
	encoded := make(map[string]string, len(settings))
	for key, value := range settings {
		if !allowedSettingKey(key) {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Unsupported setting: "+key)
		}
		value = dereferenceSettingValue(value)
		candidateRaw[key] = value
		data, err := jsonx.Marshal(value)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid setting value")
		}
		encoded[key] = string(data)
		keys = append(keys, key)
	}

	candidateLimits, err := runtimeLimitsFromRaw(candidateRaw)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	candidatePublic := settingsFromRaw(candidateRaw)
	candidatePublic.SetupCompleted = s.setupCompleted(ctx)

	if s.txManager == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Settings transaction manager is unavailable")
	}
	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start settings update")
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := s.repo.WithTx(tx)
	sort.Strings(keys)
	for _, key := range keys {
		if err := txRepo.Upsert(ctx, key, encoded[key]); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save settings")
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save settings")
	}

	s.mu.Lock()
	s.raw = candidateRaw
	s.data = candidatePublic
	s.limits = candidateLimits
	s.mu.Unlock()
	return s.Admin(ctx)
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

func runtimeLimitsFromRaw(raw map[string]any) (models.RuntimeLimits, error) {
	limits := defaultRuntimeLimits()
	fields := []struct {
		key      string
		fallback int64
		min      int64
		max      int64
		set      func(int64)
	}{
		{"limits.upload_chunk_bytes", limits.UploadChunkBytes, constants.MinRuntimeUploadChunkBytes, constants.HardMaxUploadChunkBytes, func(value int64) { limits.UploadChunkBytes = value }},
		{"limits.upload_chunks", int64(limits.UploadChunks), constants.MinRuntimeUploadChunks, constants.HardMaxUploadChunks, func(value int64) { limits.UploadChunks = int(value) }},
		{"limits.upload_sessions", int64(limits.UploadSessions), constants.MinRuntimeUploadSessions, constants.HardMaxUploadSessions, func(value int64) { limits.UploadSessions = int(value) }},
		{"limits.upload_bytes", limits.UploadBytes, constants.MinRuntimeUploadBytes, constants.HardMaxUploadBytes, func(value int64) { limits.UploadBytes = value }},
		{"limits.upload_session_ttl_seconds", limits.UploadSessionTTLSeconds, int64(constants.MinRuntimeUploadSessionTTL / time.Second), int64(constants.HardMaxUploadSessionTTL / time.Second), func(value int64) { limits.UploadSessionTTLSeconds = value }},
		{"limits.cover_bytes", limits.CoverBytes, constants.MinRuntimeCoverBytes, constants.HardMaxCoverBytes, func(value int64) { limits.CoverBytes = value }},
		{"limits.site_asset_bytes", limits.SiteAssetBytes, constants.MinRuntimeSiteAssetBytes, constants.HardMaxSiteAssetBytes, func(value int64) { limits.SiteAssetBytes = value }},
	}
	for _, field := range fields {
		value, ok := raw[field.key]
		if !ok {
			field.set(field.fallback)
			continue
		}
		integer, ok := strictInteger(value)
		if !ok || integer < field.min || integer > field.max {
			return models.RuntimeLimits{}, errors.New("Invalid runtime limit: " + field.key)
		}
		field.set(integer)
	}
	if limits.UploadChunkBytes > limits.UploadBytes || limits.UploadBytes > limits.UploadChunkBytes*int64(limits.UploadChunks) {
		return models.RuntimeLimits{}, errors.New("Invalid runtime limits: require upload chunk <= total <= chunk * chunks")
	}
	return limits, nil
}

func strictInteger(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || typed != math.Trunc(typed) || typed < math.MinInt64 || typed > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	default:
		return 0, false
	}
}

func dereferenceSettingValue(value any) any {
	switch typed := value.(type) {
	case *string:
		return *typed
	case *bool:
		return *typed
	case *int:
		return *typed
	case *int64:
		return *typed
	case *[]string:
		return *typed
	default:
		return value
	}
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
		"font.enable_custom_font_upload",
		"limits.upload_chunk_bytes",
		"limits.upload_chunks",
		"limits.upload_sessions",
		"limits.upload_bytes",
		"limits.upload_session_ttl_seconds",
		"limits.cover_bytes",
		"limits.site_asset_bytes":
		return true
	default:
		return false
	}
}

func (s *settingsService) SaveAsset(ctx context.Context, target string, fileData []byte, fileName string, urlStr string) (string, error) {
	if target != "logo" && target != "favicon" {
		return "", apperrors.New(apperrors.ErrBadRequest, "Invalid asset target")
	}
	limit := s.Limits().SiteAssetBytes
	publicDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to create public directory")
	}

	if len(fileData) > 0 {
		ext, err := bookparser.ValidateImage(fileData, limit)
		if err != nil {
			return "", apperrors.New(apperrors.ErrBadRequest, "Invalid image")
		}
		outFilename := target + ext
		destPath := filepath.Join(publicDir, outFilename)
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

		data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		if err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to read downloaded asset")
		}
		ext, err := bookparser.ValidateImage(data, constants.MaxSiteAssetBytes)
		if err != nil {
			return "", apperrors.New(apperrors.ErrBadRequest, "Downloaded asset is not a valid image")
		}
		outFilename := target + ext
		destPath := filepath.Join(publicDir, outFilename)
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to save downloaded asset")
		}
		return "/public/" + outFilename, nil
	}

	return "", apperrors.New(apperrors.ErrBadRequest, "Provide a file or URL")
}
