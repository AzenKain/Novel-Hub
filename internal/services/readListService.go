package services

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/bookparser/comic"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

type ReadListService interface {
	CreateReadList(ctx context.Context, userID string, dto request.CreateReadListDto) (*response.ReadListResponse, error)
	UpdateReadList(ctx context.Context, id string, userID string, dto request.UpdateReadListDto) (*response.ReadListResponse, error)
	DeleteReadList(ctx context.Context, id string, userID string) error
	GetUserReadLists(ctx context.Context, userID string, cursorCreatedAt *time.Time, cursorID string, limit int64) ([]*response.ReadListResponse, error)
	GetReadList(ctx context.Context, id string, userID string) (*response.ReadListResponse, error)
	GetReadListBooks(ctx context.Context, id string, userID string, claims *response.JWTClaims) ([]*response.ReadListBookResponse, error)
	AddBook(ctx context.Context, id string, userID string, bookID string, claims *response.JWTClaims) error
	RemoveBook(ctx context.Context, id string, userID string, bookID string) error
	Reorder(ctx context.Context, id string, userID string, bookIDs []string) error
	NextInOrder(ctx context.Context, id string, userID string, afterBookID string, claims *response.JWTClaims) (*response.ReadListNextResponse, error)
	ImportCBL(ctx context.Context, userID string, r io.Reader, fallbackName string) (*response.ImportCBLResponse, error)
}

type readListService struct {
	repo      repositories.ReadListRepository
	bookRepo  repositories.BookDBRepository
	bookSvc   BookService
	txManager database.TxManager
}

func NewReadListService(repo repositories.ReadListRepository, bookRepo repositories.BookDBRepository, bookSvc BookService, txManager database.TxManager) ReadListService {
	return &readListService{repo: repo, bookRepo: bookRepo, bookSvc: bookSvc, txManager: txManager}
}

func (s *readListService) requireOwnership(ctx context.Context, readListID string, userID string) error {
	owned, err := s.repo.ReadListOwnedByUser(ctx, readListID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return apperrors.New(apperrors.ErrForbidden, "Read list is not accessible")
	}
	return nil
}

func (s *readListService) CreateReadList(ctx context.Context, userID string, dto request.CreateReadListDto) (*response.ReadListResponse, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Name is required")
	}
	entity, err := s.repo.CreateReadList(ctx, uuid.Must(uuid.NewV7()).String(), userID, name, strings.TrimSpace(dto.Description))
	if err != nil {
		return nil, err
	}
	return entity.ToResponse(0), nil
}

func (s *readListService) UpdateReadList(ctx context.Context, id string, userID string, dto request.UpdateReadListDto) (*response.ReadListResponse, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Name is required")
	}
	entity, err := s.repo.UpdateReadList(ctx, id, userID, name, strings.TrimSpace(dto.Description))
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Read list not found")
		}
		return nil, err
	}
	counts, err := s.repo.CountBooksInReadLists(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return entity.ToResponse(counts[id]), nil
}

func (s *readListService) DeleteReadList(ctx context.Context, id string, userID string) error {
	if err := s.requireOwnership(ctx, id, userID); err != nil {
		return err
	}
	return s.repo.DeleteReadList(ctx, id, userID)
}

func (s *readListService) GetUserReadLists(ctx context.Context, userID string, cursorCreatedAt *time.Time, cursorID string, limit int64) ([]*response.ReadListResponse, error) {
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		limit = 50
	}
	lists, err := s.repo.GetUserReadLists(ctx, userID, cursorCreatedAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	return models.ReadListEntitiesToResponse(lists, nil), nil
}

func (s *readListService) GetReadList(ctx context.Context, id string, userID string) (*response.ReadListResponse, error) {
	if err := s.requireOwnership(ctx, id, userID); err != nil {
		return nil, err
	}
	lists, err := s.repo.GetReadListsByIDs(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	if len(lists) == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "Read list not found")
	}
	return lists[0].ToResponse(lists[0].BookCount), nil
}

// Position is the index in the stored order rather than the raw position column, so removing an
// entry never leaves the list numbered 1, 2, 4 in the UI.
func (s *readListService) GetReadListBooks(ctx context.Context, id string, userID string, claims *response.JWTClaims) ([]*response.ReadListBookResponse, error) {
	if err := s.requireOwnership(ctx, id, userID); err != nil {
		return nil, err
	}
	bookIDs, err := s.repo.GetReadListBookIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(bookIDs) == 0 {
		return []*response.ReadListBookResponse{}, nil
	}
	books, err := s.bookRepo.GetBooksByIDs(ctx, bookIDs)
	if err != nil {
		return nil, err
	}
	readable, allowed := s.bookSvc.FilterReadableBooks(ctx, books, claims)
	if !allowed {
		return nil, apperrors.New(apperrors.ErrForbidden, "Sign in to read this library")
	}
	byID := make(map[string]*models.BookEntity, len(readable))
	for _, book := range readable {
		byID[book.ID] = book
	}
	out := make([]*response.ReadListBookResponse, 0, len(bookIDs))
	for i, bookID := range bookIDs {
		book, ok := byID[bookID]
		if !ok {
			continue
		}
		out = append(out, &response.ReadListBookResponse{Position: int64(i), Book: book.ToResponse()})
	}
	return out, nil
}

func (s *readListService) AddBook(ctx context.Context, id string, userID string, bookID string, claims *response.JWTClaims) error {
	if err := s.requireOwnership(ctx, id, userID); err != nil {
		return err
	}
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return err
	}
	if !s.bookSvc.CanReadBook(ctx, book, claims) {
		return apperrors.New(apperrors.ErrForbidden, "Book is not accessible")
	}
	return s.repo.AppendBookToReadList(ctx, id, bookID)
}

func (s *readListService) RemoveBook(ctx context.Context, id string, userID string, bookID string) error {
	if err := s.requireOwnership(ctx, id, userID); err != nil {
		return err
	}
	return s.repo.RemoveBookFromReadList(ctx, id, bookID)
}

// The submitted ids must be the stored set exactly: a short array would otherwise renumber the
// entries it does contain and silently leave the rest stacked at their old positions.
func (s *readListService) Reorder(ctx context.Context, id string, userID string, bookIDs []string) error {
	if err := s.requireOwnership(ctx, id, userID); err != nil {
		return err
	}
	stored, err := s.repo.GetReadListBookIDs(ctx, id)
	if err != nil {
		return err
	}
	if len(stored) != len(bookIDs) {
		return apperrors.New(apperrors.ErrBadRequest, "The order must list every book in the read list exactly once")
	}
	remaining := make(map[string]struct{}, len(stored))
	for _, bookID := range stored {
		remaining[bookID] = struct{}{}
	}
	for _, bookID := range bookIDs {
		if _, ok := remaining[bookID]; !ok {
			return apperrors.New(apperrors.ErrBadRequest, "The order must list every book in the read list exactly once")
		}
		delete(remaining, bookID)
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.repo.WithTx(tx).ReplaceReadListOrder(ctx, id, bookIDs); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.repo.InvalidateReadListCache(ctx, id, userID)
	return nil
}

// A dead end is not an error: running past the last entry, or hitting a book the reader may not
// open, both mean "nothing more to read here" and the reader hides the button.
func (s *readListService) NextInOrder(ctx context.Context, id string, userID string, afterBookID string, claims *response.JWTClaims) (*response.ReadListNextResponse, error) {
	if err := s.requireOwnership(ctx, id, userID); err != nil {
		return nil, err
	}

	var nextID string
	var err error
	if strings.TrimSpace(afterBookID) == "" {
		nextID, err = s.repo.GetFirstInReadList(ctx, id)
	} else {
		nextID, err = s.repo.GetNextInReadList(ctx, id, afterBookID)
	}
	if err != nil {
		if apperrors.IsNotFound(err) {
			return &response.ReadListNextResponse{HasNext: false}, nil
		}
		return nil, err
	}

	candidate, err := s.bookRepo.GetBook(ctx, nextID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return &response.ReadListNextResponse{HasNext: false}, nil
		}
		return nil, err
	}
	if !s.bookSvc.CanReadBook(ctx, candidate, claims) {
		return &response.ReadListNextResponse{HasNext: false}, nil
	}

	position := int64(-1)
	if ids, err := s.repo.GetReadListBookIDs(ctx, id); err == nil {
		for i, bookID := range ids {
			if bookID == nextID {
				position = int64(i)
				break
			}
		}
	}
	return &response.ReadListNextResponse{Position: position, Book: candidate.ToResponse(), HasNext: true}, nil
}

func (s *readListService) ImportCBL(ctx context.Context, userID string, r io.Reader, fallbackName string) (*response.ImportCBLResponse, error) {
	parsed, err := comic.ParseCBL(r)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "The .cbl file could not be read: "+err.Error())
	}

	seriesKeys := make([]string, 0, len(parsed.Books))
	seenKeys := make(map[string]struct{}, len(parsed.Books))
	for _, entry := range parsed.Books {
		key := strings.ToLower(strings.TrimSpace(entry.Series))
		if key == "" {
			continue
		}
		if _, ok := seenKeys[key]; ok {
			continue
		}
		seenKeys[key] = struct{}{}
		seriesKeys = append(seriesKeys, key)
	}

	matches, err := s.repo.MatchBooksBySeriesNames(ctx, seriesKeys)
	if err != nil {
		return nil, err
	}
	bySeries := make(map[string][]repositories.SeriesIndexMatch, len(seriesKeys))
	for _, match := range matches {
		bySeries[match.SeriesKey] = append(bySeries[match.SeriesKey], match)
	}

	matchedIDs := make([]string, 0, len(parsed.Books))
	usedIDs := make(map[string]struct{}, len(parsed.Books))
	unmatched := make([]response.CBLUnmatchedEntry, 0)
	for _, entry := range parsed.Books {
		key := strings.ToLower(strings.TrimSpace(entry.Series))
		candidates := bySeries[key]
		if key == "" || len(candidates) == 0 {
			unmatched = append(unmatched, response.CBLUnmatchedEntry{Series: entry.Series, Number: entry.Number, Reason: "series not in library"})
			continue
		}
		bookID := pickIssue(candidates, entry.Number)
		if bookID == "" {
			unmatched = append(unmatched, response.CBLUnmatchedEntry{Series: entry.Series, Number: entry.Number, Reason: "issue not in library"})
			continue
		}
		if _, ok := usedIDs[bookID]; ok {
			continue
		}
		usedIDs[bookID] = struct{}{}
		matchedIDs = append(matchedIDs, bookID)
	}

	name := parsed.Name
	if name == "" {
		name = strings.TrimSpace(fallbackName)
	}
	if name == "" {
		name = "Imported reading list"
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := s.repo.WithTx(tx)

	list, err := txRepo.CreateReadList(ctx, uuid.Must(uuid.NewV7()).String(), userID, name, "")
	if err != nil {
		return nil, err
	}
	for _, bookID := range matchedIDs {
		if err := txRepo.AppendBookToReadList(ctx, list.ID, bookID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.repo.InvalidateReadListCache(ctx, list.ID, userID)

	return &response.ImportCBLResponse{
		ReadList:  list.ToResponse(int64(len(matchedIDs))),
		Total:     len(parsed.Books),
		Matched:   len(matchedIDs),
		Unmatched: unmatched,
	}, nil
}

// "1", "01" and "1.0" are the same issue to every human and to none of them is it the same string,
// so numeric comparison comes first and the literal match is only the fallback for "1A"-style runs.
func pickIssue(candidates []repositories.SeriesIndexMatch, number string) string {
	wanted := strings.TrimSpace(number)
	if wanted == "" {
		if len(candidates) == 1 {
			return candidates[0].BookID
		}
		return ""
	}
	wantedFloat, wantedIsNumber := parseIssueNumber(wanted)
	for _, candidate := range candidates {
		have := strings.TrimSpace(candidate.SeriesIndex)
		if have == wanted {
			return candidate.BookID
		}
		if wantedIsNumber {
			if haveFloat, ok := parseIssueNumber(have); ok && haveFloat == wantedFloat {
				return candidate.BookID
			}
		}
		if strings.EqualFold(have, wanted) {
			return candidate.BookID
		}
	}
	return ""
}

func parseIssueNumber(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
