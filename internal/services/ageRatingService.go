package services

import (
	"context"
	"regexp"

	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
)

var numeric6DigitRegex = regexp.MustCompile(`^\d{6}$`)

type AgeRatingService interface {
	GetContentWarnings(ctx context.Context) ([]*response.ContentWarningResponse, error)
	GetBookContentWarnings(ctx context.Context, bookID string) ([]*response.ContentWarningResponse, error)
	UpdateBookAgeRating(ctx context.Context, bookID string, dto *request.UpdateBookAgeRatingDto) error
	GetKidsModeInfo(ctx context.Context, userID string) (*response.KidsModeInfoResponse, error)
	SetKidsModePin(ctx context.Context, userID string, dto *request.SetKidsModePinDto) error
	ToggleKidsMode(ctx context.Context, userID string, dto *request.ToggleKidsModeDto) error
	IsAgeAllowed(bookRating string, maxAllowed string) bool
}

type ageRatingService struct {
	repo repositories.AgeRatingRepository
}

func NewAgeRatingService(repo repositories.AgeRatingRepository) AgeRatingService {
	return &ageRatingService{repo: repo}
}

func (s *ageRatingService) IsAgeAllowed(bookRating string, maxAllowed string) bool {
	if maxAllowed == "" {
		maxAllowed = constants.AgeRatingR18
	}
	bookLevel := constants.AgeRatingLevels[bookRating]
	if bookLevel == 0 {
		bookLevel = constants.AgeRatingLevels[constants.AgeRatingG]
	}
	maxLevel := constants.AgeRatingLevels[maxAllowed]
	if maxLevel == 0 {
		maxLevel = constants.AgeRatingLevels[constants.AgeRatingR18]
	}
	return bookLevel <= maxLevel
}

func (s *ageRatingService) GetContentWarnings(ctx context.Context) ([]*response.ContentWarningResponse, error) {
	entities, err := s.repo.GetContentWarnings(ctx)
	if err != nil {
		return nil, err
	}
	return models.ContentWarningsToResponse(entities), nil
}

func (s *ageRatingService) GetBookContentWarnings(ctx context.Context, bookID string) ([]*response.ContentWarningResponse, error) {
	if bookID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Book ID is required")
	}
	entities, err := s.repo.GetBookContentWarnings(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return models.ContentWarningsToResponse(entities), nil
}

func (s *ageRatingService) UpdateBookAgeRating(ctx context.Context, bookID string, dto *request.UpdateBookAgeRatingDto) error {
	if dto == nil || dto.AgeRating == "" {
		return apperrors.New(apperrors.ErrBadRequest, "Age rating is required")
	}
	if _, ok := constants.AgeRatingLevels[dto.AgeRating]; !ok {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid age rating classification")
	}

	warningIDs := dto.ContentWarningIDs
	if warningIDs == nil {
		warningIDs = []string{}
	}

	return s.repo.UpdateBookAgeRatingAndWarnings(ctx, bookID, dto.AgeRating, warningIDs)
}

func (s *ageRatingService) GetKidsModeInfo(ctx context.Context, userID string) (*response.KidsModeInfoResponse, error) {
	info, err := s.repo.GetUserKidsModeInfo(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User kids mode info not found")
	}
	return info.ToResponse(), nil
}

func (s *ageRatingService) SetKidsModePin(ctx context.Context, userID string, dto *request.SetKidsModePinDto) error {
	if dto == nil || !numeric6DigitRegex.MatchString(dto.Pin) {
		return apperrors.New(apperrors.ErrBadRequest, "Kids mode PIN must be exactly 6 digits")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Pin), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to hash PIN")
	}

	return s.repo.SetKidsModePin(ctx, userID, string(hashed))
}

func (s *ageRatingService) ToggleKidsMode(ctx context.Context, userID string, dto *request.ToggleKidsModeDto) error {
	if dto == nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid payload")
	}

	info, err := s.repo.GetUserKidsModeInfo(ctx, userID)
	if err != nil || info == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	// Enabling Kids Mode: require a 6-digit PIN to already be set
	if dto.Enable {
		if info.KidsModePinHash == nil || *info.KidsModePinHash == "" {
			return apperrors.New(apperrors.ErrBadRequest, "Please set a 6-digit PIN before enabling Kids Mode")
		}
		return s.repo.SetKidsModeStatus(ctx, userID, true)
	}

	// Disabling Kids Mode: MUST verify 6-digit PIN via bcrypt
	if info.KidsModePinHash == nil || *info.KidsModePinHash == "" {
		return s.repo.SetKidsModeStatus(ctx, userID, false)
	}

	if !numeric6DigitRegex.MatchString(dto.Pin) {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid 6-digit PIN format")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*info.KidsModePinHash), []byte(dto.Pin)); err != nil {
		return apperrors.New(apperrors.ErrForbidden, "Incorrect 6-digit PIN")
	}

	return s.repo.SetKidsModeStatus(ctx, userID, false)
}
