package services

import (
	"context"
	"slices"
	"strings"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/constants"
)

func isGuestClaims(c *response.JWTClaims) bool {
	return c == nil || c.UId == "0" || slices.Contains(c.Roles, constants.RoleTypeGuest)
}

func resolveClaims(claims *response.JWTClaims) *response.JWTClaims {
	if claims == nil {
		return &response.JWTClaims{
			UId:   "0",
			Roles: []constants.RoleType{constants.RoleTypeGuest},
		}
	}
	return claims
}

func (s *bookService) FilterReadableBooks(ctx context.Context, books []*models.BookEntity, claims *response.JWTClaims) ([]*models.BookEntity, bool) {
	if len(books) == 0 {
		return books, true
	}
	c := resolveClaims(claims)
	if isGuestClaims(c) {
		settings, err := s.settings.Public(ctx)
		if err == nil && settings.GuestLoginRequired {
			return nil, false
		}
	}
	out := make([]*models.BookEntity, 0, len(books))
	for _, book := range books {
		if book != nil && s.CanReadBook(ctx, book, c) {
			out = append(out, book)
		}
	}
	return out, true
}

func (s *bookService) CanReadBook(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	if book == nil {
		return false
	}
	c := resolveClaims(claims)
	if isGuestClaims(c) && !s.settings.GuestAllows(book.LibraryID) {
		return false
	}
	if s.permissions.IsAdmin(c.RoleIDs, c.Roles) {
		return true
	}
	return s.permissions.CanRoles(c.RoleIDs, c.Roles, constants.PermBookRead, map[string]any{"library_id": book.LibraryID})
}

func (s *bookService) CanDownloadBook(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	if book == nil {
		return false
	}
	c := resolveClaims(claims)
	if isGuestClaims(c) && !s.settings.GuestAllows(book.LibraryID) {
		return false
	}
	if s.permissions.IsAdmin(c.RoleIDs, c.Roles) {
		return true
	}
	return s.permissions.CanRoles(c.RoleIDs, c.Roles, constants.PermBookDownload, map[string]any{"library_id": book.LibraryID})
}

func (s *bookService) CanUpdateBook(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	if book == nil {
		return false
	}
	c := resolveClaims(claims)
	if s.permissions.IsAdmin(c.RoleIDs, c.Roles) {
		return true
	}
	return s.permissions.CanRoles(c.RoleIDs, c.Roles, constants.PermBookEdit, map[string]any{"library_id": book.LibraryID})
}

func (s *bookService) CanDeleteBook(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	if book == nil {
		return false
	}
	c := resolveClaims(claims)
	if s.permissions.IsAdmin(c.RoleIDs, c.Roles) {
		return true
	}
	return s.permissions.CanRoles(c.RoleIDs, c.Roles, constants.PermBookDelete, map[string]any{"library_id": book.LibraryID})
}

func (s *bookService) SafeDownloadFilename(title string, ext string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		name = "book"
	}
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, name)
	name = strings.Trim(name, " .-")
	if name == "" {
		name = "book"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return name + ext
}
