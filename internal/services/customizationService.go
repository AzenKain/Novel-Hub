package services

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/localfs"
	"novelhub/pkg/netx"
)

type CustomizationService interface {
	ListSoundscapes(ctx context.Context, claims *response.JWTClaims, cursor *time.Time, cursorID string, limit int64) ([]*response.SoundscapeResponse, *string, error)
	GetSoundscape(ctx context.Context, id string, claims *response.JWTClaims) (*models.SoundscapeEntity, error)
	CreateSoundscape(ctx context.Context, claims *response.JWTClaims, dto *request.UploadSoundscapeDto, fileData []byte, originalFilename string) (*response.SoundscapeResponse, error)
	DeleteSoundscape(ctx context.Context, id string, claims *response.JWTClaims) error
	GetSoundscapeFilePath(ctx context.Context, id string, claims *response.JWTClaims) (string, string, error)

	ListCustomFonts(ctx context.Context, claims *response.JWTClaims, cursor *time.Time, cursorID string, limit int64) ([]*response.CustomFontResponse, *string, error)
	GetCustomFont(ctx context.Context, id string, claims *response.JWTClaims) (*models.CustomFontEntity, error)
	CreateCustomFont(ctx context.Context, claims *response.JWTClaims, dto *request.UploadFontDto, fileData []byte, originalFilename string) (*response.CustomFontResponse, error)
	DeleteCustomFont(ctx context.Context, id string, claims *response.JWTClaims) error
	GetFontFilePath(ctx context.Context, id string, claims *response.JWTClaims) (string, string, error)

	ListCustomThemes(ctx context.Context, claims *response.JWTClaims, cursor *time.Time, cursorID string, limit int64) ([]*response.CustomThemeResponse, *string, error)
	GetCustomTheme(ctx context.Context, id string, claims *response.JWTClaims) (*response.CustomThemeResponse, error)
	CreateCustomTheme(ctx context.Context, claims *response.JWTClaims, dto *request.CreateCustomThemeDto) (*response.CustomThemeResponse, error)
	UpdateCustomTheme(ctx context.Context, id string, claims *response.JWTClaims, dto *request.UpdateCustomThemeDto) (*response.CustomThemeResponse, error)
	DeleteCustomTheme(ctx context.Context, id string, claims *response.JWTClaims) error
}

type customizationService struct {
	repo            repositories.CustomizationRepository
	permissionCache PermissionCache
	settings        SettingsService
	dataDir         string
}

func NewCustomizationService(
	repo repositories.CustomizationRepository,
	permissionCache PermissionCache,
	settings SettingsService,
	dataDir string,
) CustomizationService {
	soundDir := filepath.Join(dataDir, "soundscapes")
	fontDir := filepath.Join(dataDir, "fonts")
	_ = os.MkdirAll(soundDir, 0755)
	_ = os.MkdirAll(fontDir, 0755)

	return &customizationService{
		repo:            repo,
		permissionCache: permissionCache,
		settings:        settings,
		dataDir:         dataDir,
	}
}

func (s *customizationService) ListSoundscapes(ctx context.Context, claims *response.JWTClaims, cursor *time.Time, cursorID string, limit int64) ([]*response.SoundscapeResponse, *string, error) {
	var userID *string
	if claims != nil && claims.UId != "" {
		userID = &claims.UId
	}
	entities, err := s.repo.ListSoundscapes(ctx, userID, cursor, cursorID, limit)
	if err != nil {
		return nil, nil, apperrors.New(apperrors.ErrInternalError, "Failed to list soundscapes")
	}

	result := make([]*response.SoundscapeResponse, len(entities))
	for i, e := range entities {
		result[i] = e.ToResponse()
	}

	var nextCursor *string
	if int64(len(entities)) == limit && len(entities) > 0 {
		last := entities[len(entities)-1]
		if t, err := time.Parse("2006-01-02 15:04:05", last.UpdatedAt); err == nil {
			enc := convert.EncodeCursor(t, last.ID)
			nextCursor = &enc
		} else if t, err := time.Parse(time.RFC3339Nano, last.UpdatedAt); err == nil {
			enc := convert.EncodeCursor(t, last.ID)
			nextCursor = &enc
		}
	}

	return result, nextCursor, nil
}

func (s *customizationService) GetSoundscape(ctx context.Context, id string, claims *response.JWTClaims) (*models.SoundscapeEntity, error) {
	e, err := s.repo.GetSoundscapeByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch soundscape")
	}
	if e == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Soundscape not found")
	}
	if !e.IsSystem {
		if claims == nil || claims.UId == "" {
			return nil, apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
		}
		isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminSoundscapeManage, nil))
		if !isAdmin && (e.UserID == nil || *e.UserID != claims.UId) {
			return nil, apperrors.New(apperrors.ErrForbidden, "Access denied to private soundscape")
		}
	}
	return e, nil
}

func (s *customizationService) CreateSoundscape(
	ctx context.Context,
	claims *response.JWTClaims,
	dto *request.UploadSoundscapeDto,
	fileData []byte,
	originalFilename string,
) (*response.SoundscapeResponse, error) {
	if claims == nil || claims.UId == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}

	isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminSoundscapeManage, nil))
	if dto.IsSystem && !isAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only administrators can create system soundscapes")
	}

	if !dto.IsSystem && !isAdmin && s.permissionCache != nil && !s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermUserSoundscapeManage, nil) {
		return nil, apperrors.New(apperrors.ErrForbidden, "You do not have permission to upload personal soundscapes")
	}

	if len(fileData) == 0 && dto.AudioURL != "" {
		client := netx.NewSafeHTTPClient(30 * time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, dto.AudioURL, nil)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid audio URL")
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Failed to download audio from URL: "+err.Error())
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("Download failed with HTTP %d", resp.StatusCode))
		}

		maxSoundscapeBytes := int64(constants.MaxSoundscapeBytes)
		if s.settings != nil {
			maxSoundscapeBytes = s.settings.Limits().SoundscapeBytes
		}
		limitReader := io.LimitReader(resp.Body, maxSoundscapeBytes+1)
		data, err := io.ReadAll(limitReader)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to read downloaded audio")
		}
		if int64(len(data)) > maxSoundscapeBytes {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Audio file exceeds configured size limit")
		}
		fileData = data
		if originalFilename == "" {
			originalFilename = filepath.Base(dto.AudioURL)
		}
	}

	if len(fileData) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "No audio file or URL provided")
	}

	ext, err := validateAudioFormatAndMagic(fileData, originalFilename)
	if err != nil {
		return nil, err
	}

	soundID := uuid.NewString()
	storedFilename := fmt.Sprintf("%s%s", soundID, ext)
	soundDirPath := filepath.Join(s.dataDir, "soundscapes")
	fullFilePath, err := localfs.SafeJoin(soundDirPath, storedFilename)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Invalid storage path")
	}

	if err := os.WriteFile(fullFilePath, fileData, 0644); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save audio file to disk")
	}

	isSystemVal := int64(0)
	var userIDVal sql.NullString
	if dto.IsSystem {
		isSystemVal = 1
	} else {
		userIDVal = sql.NullString{String: claims.UId, Valid: true}
	}

	icon := dto.Icon
	if icon == "" {
		icon = "Music"
	}
	category := dto.Category
	if category == "" {
		category = "ambient"
	}
	vol := dto.Volume
	if vol <= 0 {
		vol = 0.5
	}

	created, err := s.repo.CreateSoundscape(ctx, sqlc.CreateSoundscapeParams{
		ID:       soundID,
		UserID:   userIDVal,
		Name:     dto.Name,
		Category: category,
		FilePath: fullFilePath,
		Icon:     icon,
		Volume:   vol,
		IsSystem: isSystemVal,
	})
	if err != nil {
		_ = os.Remove(fullFilePath)
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save soundscape record")
	}

	return created.ToResponse(), nil
}

func (s *customizationService) DeleteSoundscape(ctx context.Context, id string, claims *response.JWTClaims) error {
	if claims == nil || claims.UId == "" {
		return apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}

	existing, err := s.repo.GetSoundscapeByID(ctx, id)
	if err != nil || existing == nil {
		return apperrors.New(apperrors.ErrNotFound, "Soundscape not found")
	}

	isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminSoundscapeManage, nil))
	if existing.IsSystem && !isAdmin {
		return apperrors.New(apperrors.ErrForbidden, "Only administrators can delete system soundscapes")
	}
	if !existing.IsSystem && !isAdmin {
		if existing.UserID == nil || *existing.UserID != claims.UId {
			return apperrors.New(apperrors.ErrForbidden, "You can only delete your own soundscapes")
		}
	}

	if existing.FilePath != "" {
		_ = os.Remove(existing.FilePath)
	}

	if err := s.repo.DeleteSoundscape(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete soundscape")
	}
	return nil
}

func (s *customizationService) GetSoundscapeFilePath(ctx context.Context, id string, claims *response.JWTClaims) (string, string, error) {
	existing, err := s.repo.GetSoundscapeByID(ctx, id)
	if err != nil || existing == nil {
		return "", "", apperrors.New(apperrors.ErrNotFound, "Soundscape not found")
	}

	if !existing.IsSystem {
		if claims == nil || claims.UId == "" {
			return "", "", apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
		}
		isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminSoundscapeManage, nil))
		if !isAdmin && (existing.UserID == nil || *existing.UserID != claims.UId) {
			return "", "", apperrors.New(apperrors.ErrForbidden, "Access denied to private soundscape")
		}
	}

	stat, err := os.Lstat(existing.FilePath)
	if err != nil || stat.IsDir() || stat.Mode()&os.ModeSymlink != 0 {
		return "", "", apperrors.New(apperrors.ErrNotFound, "Soundscape file not found on disk")
	}

	return existing.FilePath, existing.Name, nil
}

func (s *customizationService) ListCustomFonts(ctx context.Context, claims *response.JWTClaims, cursor *time.Time, cursorID string, limit int64) ([]*response.CustomFontResponse, *string, error) {
	var userID *string
	if claims != nil && claims.UId != "" {
		userID = &claims.UId
	}
	entities, err := s.repo.ListCustomFonts(ctx, userID, cursor, cursorID, limit)
	if err != nil {
		return nil, nil, apperrors.New(apperrors.ErrInternalError, "Failed to list custom fonts")
	}

	result := make([]*response.CustomFontResponse, len(entities))
	for i, e := range entities {
		result[i] = e.ToResponse()
	}

	var nextCursor *string
	if int64(len(entities)) == limit && len(entities) > 0 {
		last := entities[len(entities)-1]
		if t, err := time.Parse("2006-01-02 15:04:05", last.UpdatedAt); err == nil {
			enc := convert.EncodeCursor(t, last.ID)
			nextCursor = &enc
		} else if t, err := time.Parse(time.RFC3339Nano, last.UpdatedAt); err == nil {
			enc := convert.EncodeCursor(t, last.ID)
			nextCursor = &enc
		}
	}

	return result, nextCursor, nil
}

func (s *customizationService) GetCustomFont(ctx context.Context, id string, claims *response.JWTClaims) (*models.CustomFontEntity, error) {
	e, err := s.repo.GetCustomFontByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch custom font")
	}
	if e == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Custom font not found")
	}
	if !e.IsSystem {
		if claims == nil || claims.UId == "" {
			return nil, apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
		}
		isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminFontManage, nil))
		if !isAdmin && (e.UserID == nil || *e.UserID != claims.UId) {
			return nil, apperrors.New(apperrors.ErrForbidden, "Access denied to private custom font")
		}
	}
	return e, nil
}

func (s *customizationService) CreateCustomFont(
	ctx context.Context,
	claims *response.JWTClaims,
	dto *request.UploadFontDto,
	fileData []byte,
	originalFilename string,
) (*response.CustomFontResponse, error) {
	if claims == nil || claims.UId == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}

	isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminFontManage, nil))
	if dto.IsSystem && !isAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only administrators can create system fonts")
	}

	if !dto.IsSystem && !isAdmin && s.permissionCache != nil && !s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermUserFontManage, nil) {
		return nil, apperrors.New(apperrors.ErrForbidden, "You do not have permission to upload personal custom fonts")
	}

	fontID := uuid.NewString()
	filePath := ""

	if dto.SourceType == "file" {
		if len(fileData) == 0 {
			return nil, apperrors.New(apperrors.ErrBadRequest, "No font file uploaded")
		}
		ext, err := validateFontFormatAndMagic(fileData, originalFilename)
		if err != nil {
			return nil, err
		}
		storedFilename := fmt.Sprintf("%s%s", fontID, ext)
		fontDirPath := filepath.Join(s.dataDir, "fonts")
		fullFilePath, err := localfs.SafeJoin(fontDirPath, storedFilename)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Invalid storage path")
		}

		if err := os.WriteFile(fullFilePath, fileData, 0644); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save font file to disk")
		}
		filePath = fullFilePath
	} else if dto.SourceType == "url" {
		if dto.FontURL == "" {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Font URL is required for url source type")
		}
	} else {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid source type (must be 'file' or 'url')")
	}

	isSystemVal := int64(0)
	var userIDVal sql.NullString
	if dto.IsSystem {
		isSystemVal = 1
	} else {
		userIDVal = sql.NullString{String: claims.UId, Valid: true}
	}

	fontFamily := strings.TrimSpace(dto.FontFamily)
	if fontFamily == "" {
		fontFamily = strings.TrimSpace(dto.Name)
	}

	created, err := s.repo.CreateCustomFont(ctx, sqlc.CreateCustomFontParams{
		ID:         fontID,
		UserID:     userIDVal,
		Name:       dto.Name,
		FontFamily: fontFamily,
		SourceType: dto.SourceType,
		FilePath:   filePath,
		FontUrl:    dto.FontURL,
		IsSystem:   isSystemVal,
	})
	if err != nil {
		if filePath != "" {
			_ = os.Remove(filePath)
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save custom font record")
	}

	return created.ToResponse(), nil
}

func (s *customizationService) DeleteCustomFont(ctx context.Context, id string, claims *response.JWTClaims) error {
	if claims == nil || claims.UId == "" {
		return apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}

	existing, err := s.repo.GetCustomFontByID(ctx, id)
	if err != nil || existing == nil {
		return apperrors.New(apperrors.ErrNotFound, "Custom font not found")
	}

	isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminFontManage, nil))
	if existing.IsSystem && !isAdmin {
		return apperrors.New(apperrors.ErrForbidden, "Only administrators can delete system fonts")
	}
	if !existing.IsSystem && !isAdmin {
		if existing.UserID == nil || *existing.UserID != claims.UId {
			return apperrors.New(apperrors.ErrForbidden, "You can only delete your own fonts")
		}
	}

	if existing.FilePath != "" {
		_ = os.Remove(existing.FilePath)
	}

	if err := s.repo.DeleteCustomFont(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete custom font")
	}
	return nil
}

func (s *customizationService) GetFontFilePath(ctx context.Context, id string, claims *response.JWTClaims) (string, string, error) {
	existing, err := s.repo.GetCustomFontByID(ctx, id)
	if err != nil || existing == nil {
		return "", "", apperrors.New(apperrors.ErrNotFound, "Custom font not found")
	}

	if !existing.IsSystem {
		if claims == nil || claims.UId == "" {
			return "", "", apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
		}
		isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminFontManage, nil))
		if !isAdmin && (existing.UserID == nil || *existing.UserID != claims.UId) {
			return "", "", apperrors.New(apperrors.ErrForbidden, "Access denied to private custom font")
		}
	}

	stat, err := os.Lstat(existing.FilePath)
	if err != nil || stat.IsDir() || stat.Mode()&os.ModeSymlink != 0 {
		return "", "", apperrors.New(apperrors.ErrNotFound, "Font file not found on disk")
	}

	return existing.FilePath, existing.Name, nil
}

func (s *customizationService) ListCustomThemes(ctx context.Context, claims *response.JWTClaims, cursor *time.Time, cursorID string, limit int64) ([]*response.CustomThemeResponse, *string, error) {
	var userID *string
	if claims != nil && claims.UId != "" {
		userID = &claims.UId
	}
	entities, err := s.repo.ListCustomThemes(ctx, userID, cursor, cursorID, limit)
	if err != nil {
		return nil, nil, apperrors.New(apperrors.ErrInternalError, "Failed to list custom themes")
	}

	result := make([]*response.CustomThemeResponse, len(entities))
	for i, e := range entities {
		result[i] = e.ToResponse()
	}

	var nextCursor *string
	if int64(len(entities)) == limit && len(entities) > 0 {
		last := entities[len(entities)-1]
		if t, err := time.Parse("2006-01-02 15:04:05", last.UpdatedAt); err == nil {
			enc := convert.EncodeCursor(t, last.ID)
			nextCursor = &enc
		} else if t, err := time.Parse(time.RFC3339Nano, last.UpdatedAt); err == nil {
			enc := convert.EncodeCursor(t, last.ID)
			nextCursor = &enc
		}
	}

	return result, nextCursor, nil
}

func (s *customizationService) GetCustomTheme(ctx context.Context, id string, claims *response.JWTClaims) (*response.CustomThemeResponse, error) {
	e, err := s.repo.GetCustomThemeByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch custom theme")
	}
	if e == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Custom theme not found")
	}
	if !e.IsSystem {
		if claims == nil || claims.UId == "" {
			return nil, apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
		}
		isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminThemeManage, nil))
		if !isAdmin && (e.UserID == nil || *e.UserID != claims.UId) {
			return nil, apperrors.New(apperrors.ErrForbidden, "Access denied to private custom theme")
		}
	}
	return e.ToResponse(), nil
}

func (s *customizationService) CreateCustomTheme(
	ctx context.Context,
	claims *response.JWTClaims,
	dto *request.CreateCustomThemeDto,
) (*response.CustomThemeResponse, error) {
	if claims == nil || claims.UId == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}

	isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminThemeManage, nil))
	if dto.IsSystem && !isAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only administrators can create system themes")
	}

	if !dto.IsSystem && !isAdmin && s.permissionCache != nil && !s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermUserThemeManage, nil) {
		return nil, apperrors.New(apperrors.ErrForbidden, "You do not have permission to create personal custom themes")
	}

	if err := validateCustomCSS(dto.CustomCss); err != nil {
		return nil, err
	}

	themeID := uuid.NewString()
	isSystemVal := int64(0)
	var userIDVal sql.NullString
	if dto.IsSystem {
		isSystemVal = 1
	} else {
		userIDVal = sql.NullString{String: claims.UId, Valid: true}
	}

	created, err := s.repo.CreateCustomTheme(ctx, sqlc.CreateCustomThemeParams{
		ID:          themeID,
		UserID:      userIDVal,
		Name:        dto.Name,
		BgColor:     dto.BgColor,
		TextColor:   dto.TextColor,
		AccentColor: dto.AccentColor,
		CustomCss:   dto.CustomCss,
		IsSystem:    isSystemVal,
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create custom theme")
	}

	return created.ToResponse(), nil
}

func (s *customizationService) UpdateCustomTheme(
	ctx context.Context,
	id string,
	claims *response.JWTClaims,
	dto *request.UpdateCustomThemeDto,
) (*response.CustomThemeResponse, error) {
	if claims == nil || claims.UId == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}

	existing, err := s.repo.GetCustomThemeByID(ctx, id)
	if err != nil || existing == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Custom theme not found")
	}

	isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminThemeManage, nil))
	if existing.IsSystem && !isAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only administrators can update system themes")
	}
	if !existing.IsSystem && !isAdmin {
		if existing.UserID == nil || *existing.UserID != claims.UId {
			return nil, apperrors.New(apperrors.ErrForbidden, "You can only update your own custom themes")
		}
	}

	if err := validateCustomCSS(dto.CustomCss); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateCustomTheme(ctx, sqlc.UpdateCustomThemeParams{
		ID:          id,
		Name:        dto.Name,
		BgColor:     dto.BgColor,
		TextColor:   dto.TextColor,
		AccentColor: dto.AccentColor,
		CustomCss:   dto.CustomCss,
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update custom theme")
	}

	return updated.ToResponse(), nil
}

func (s *customizationService) DeleteCustomTheme(ctx context.Context, id string, claims *response.JWTClaims) error {
	if claims == nil || claims.UId == "" {
		return apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}

	existing, err := s.repo.GetCustomThemeByID(ctx, id)
	if err != nil || existing == nil {
		return apperrors.New(apperrors.ErrNotFound, "Custom theme not found")
	}

	isAdmin := s.permissionCache != nil && (s.permissionCache.IsAdmin(claims.RoleIDs, claims.Roles) || s.permissionCache.CanRoles(claims.RoleIDs, claims.Roles, constants.PermAdminThemeManage, nil))
	if existing.IsSystem && !isAdmin {
		return apperrors.New(apperrors.ErrForbidden, "Only administrators can delete system themes")
	}
	if !existing.IsSystem && !isAdmin {
		if existing.UserID == nil || *existing.UserID != claims.UId {
			return apperrors.New(apperrors.ErrForbidden, "You can only delete your own custom themes")
		}
	}

	if err := s.repo.DeleteCustomTheme(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete custom theme")
	}
	return nil
}

func validateAudioFormatAndMagic(data []byte, filename string) (string, error) {
	if len(data) < 4 {
		return "", apperrors.New(apperrors.ErrBadRequest, "Audio file is empty or corrupted")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp3":
		if !bytes.HasPrefix(data, []byte("ID3")) && !(len(data) >= 2 && data[0] == 0xFF && (data[1]&0xE0) == 0xE0) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match MP3 audio format")
		}
	case ".ogg":
		if !bytes.HasPrefix(data, []byte("OggS")) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match OGG audio format")
		}
	case ".wav":
		if !bytes.HasPrefix(data, []byte("RIFF")) || len(data) < 12 || !bytes.Equal(data[8:12], []byte("WAVE")) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match WAV audio format")
		}
	case ".flac":
		if !bytes.HasPrefix(data, []byte("fLaC")) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match FLAC audio format")
		}
	case ".m4a":
		if len(data) < 8 || (!bytes.Equal(data[4:8], []byte("ftyp")) && !bytes.HasPrefix(data, []byte{0x00, 0x00, 0x00})) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match M4A audio format")
		}
	case ".aac":
		if !bytes.HasPrefix(data, []byte{0xFF, 0xF1}) && !bytes.HasPrefix(data, []byte{0xFF, 0xF9}) && !bytes.HasPrefix(data, []byte("ID3")) && !(len(data) >= 8 && bytes.Equal(data[4:8], []byte("ftyp"))) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match AAC audio format")
		}
	default:
		return "", apperrors.New(apperrors.ErrBadRequest, "Unsupported audio extension. Allowed: .mp3, .ogg, .wav, .m4a, .aac, .flac")
	}
	return ext, nil
}

func validateFontFormatAndMagic(data []byte, filename string) (string, error) {
	if len(data) < 4 {
		return "", apperrors.New(apperrors.ErrBadRequest, "Font file is empty or corrupted")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".woff2":
		if !bytes.HasPrefix(data, []byte("wOF2")) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match WOFF2 font format")
		}
	case ".woff":
		if !bytes.HasPrefix(data, []byte("wOFF")) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match WOFF font format")
		}
	case ".ttf":
		if !bytes.HasPrefix(data, []byte{0x00, 0x01, 0x00, 0x00}) && !bytes.HasPrefix(data, []byte("true")) && !bytes.HasPrefix(data, []byte("typ1")) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match TTF font format")
		}
	case ".otf":
		if !bytes.HasPrefix(data, []byte("OTTO")) && !bytes.HasPrefix(data, []byte{0x00, 0x01, 0x00, 0x00}) {
			return "", apperrors.New(apperrors.ErrBadRequest, "File content does not match OTF font format")
		}
	default:
		return "", apperrors.New(apperrors.ErrBadRequest, "Unsupported font extension. Allowed: .woff2, .woff, .ttf, .otf")
	}
	return ext, nil
}

func validateCustomCSS(css string) error {
	if css == "" {
		return nil
	}
	lower := strings.ToLower(css)
	dangerous := []string{
		"<script", "</script", "<style", "</style", "<svg", "</svg",
		"<iframe", "<object", "<embed", "<link", "<img", "<body", "<html",
		"javascript:", "vbscript:", "expression(", "-moz-binding", "@import", "behavior:",
	}
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			return apperrors.New(apperrors.ErrBadRequest, "Custom CSS contains disallowed or dangerous syntax ("+d+")")
		}
	}
	return nil
}
