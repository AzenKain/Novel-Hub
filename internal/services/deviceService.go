package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
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
	"novelhub/pkg/jsonx"
	"novelhub/pkg/mailer"
	"novelhub/pkg/netx"
	"novelhub/pkg/worker"
)

type DeviceService interface {
	CreateDevice(ctx context.Context, userID string, dto *request.CreateUserDeviceDto) (*response.UserDeviceResponse, error)
	ListDevices(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]*response.UserDeviceResponse, error)
	DeleteDevice(ctx context.Context, id string, userID string) error
	PushBookToDevice(ctx context.Context, userID string, bookID string, deviceID string, claims *response.JWTClaims) error
	ExecutePushJob(ctx context.Context, payloadJSON string) error
}

type deviceService struct {
	repo        repositories.DeviceRepository
	bookRepo    repositories.BookDBRepository
	bookSvc     BookService
	settingsSvc SettingsService
	permissions PermissionCache
	jobQueue    *worker.Queue
}

func NewDeviceService(
	repo repositories.DeviceRepository,
	bookRepo repositories.BookDBRepository,
	bookSvc BookService,
	settingsSvc SettingsService,
	permissions PermissionCache,
	jobQueue *worker.Queue,
) DeviceService {
	return &deviceService{
		repo:        repo,
		bookRepo:    bookRepo,
		bookSvc:     bookSvc,
		settingsSvc: settingsSvc,
		permissions: permissions,
		jobQueue:    jobQueue,
	}
}

func (s *deviceService) CreateDevice(ctx context.Context, userID string, dto *request.CreateUserDeviceDto) (*response.UserDeviceResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}

	entity, err := s.repo.Create(ctx, sqlc.CreateUserDeviceParams{
		ID:            uuid.NewString(),
		UserID:        userID,
		Name:          strings.TrimSpace(dto.Name),
		DeviceType:    strings.TrimSpace(strings.ToLower(dto.DeviceType)),
		TargetAddress: strings.TrimSpace(dto.TargetAddress),
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create device")
	}

	return entity.ToResponse(), nil
}

func (s *deviceService) ListDevices(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]*response.UserDeviceResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}

	entities, err := s.repo.ListByUserID(ctx, userID, cursor, cursorID, limit)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to list devices")
	}

	return models.UserDeviceEntitiesToResponse(entities), nil
}

func (s *deviceService) DeleteDevice(ctx context.Context, id string, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}

	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete device")
	}

	return nil
}

type pushJobPayload struct {
	UserID   string `json:"user_id"`
	BookID   string `json:"book_id"`
	DeviceID string `json:"device_id"`
}

func (s *deviceService) PushBookToDevice(ctx context.Context, userID string, bookID string, deviceID string, claims *response.JWTClaims) error {
	if strings.TrimSpace(userID) == "" {
		return apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}

	device, err := s.repo.GetByID(ctx, deviceID)
	if err != nil || device == nil || device.UserID != userID {
		return apperrors.New(apperrors.ErrNotFound, "Device not found")
	}

	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return apperrors.New(apperrors.ErrNotFound, "Book not found")
	}

	if s.bookSvc != nil && !s.bookSvc.CanDownloadBook(ctx, book, claims) {
		return apperrors.New(apperrors.ErrForbidden, "Download permission denied")
	}

	payload, err := jsonx.MarshalString(pushJobPayload{
		UserID:   userID,
		BookID:   bookID,
		DeviceID: deviceID,
	})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal push payload")
	}

	if s.jobQueue != nil {
		jobID := uuid.Must(uuid.NewV7()).String()
		if err := s.jobQueue.Enqueue(ctx, worker.Job{
			ID:      jobID,
			Type:    "push_book_to_device",
			Payload: payload,
		}); err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to enqueue device push job")
		}
		return nil
	}

	return s.ExecutePushJob(ctx, payload)
}

func (s *deviceService) ExecutePushJob(ctx context.Context, payloadJSON string) error {
	var payload pushJobPayload
	if err := jsonx.UnmarshalString(payloadJSON, &payload); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid push payload")
	}

	device, err := s.repo.GetByID(ctx, payload.DeviceID)
	if err != nil || device == nil {
		return apperrors.New(apperrors.ErrNotFound, "Target device not found")
	}

	book, err := s.bookRepo.GetBook(ctx, payload.BookID)
	if err != nil || book == nil {
		return apperrors.New(apperrors.ErrNotFound, "Target book not found")
	}

	files, err := s.bookRepo.GetFilesByBookId(ctx, payload.BookID)
	if err != nil || len(files) == 0 {
		return apperrors.New(apperrors.ErrNotFound, "No files found for target book")
	}

	targetFile := files[0]
	for _, f := range files {
		if strings.EqualFold(f.Format, "epub") {
			targetFile = f
			break
		}
	}

	filePath := targetFile.Path

	switch device.DeviceType {
	case "kindle", "pocketbook":
		smtpConfig, err := s.settingsSvc.SMTP(ctx)
		if err != nil {
			return apperrors.New(apperrors.ErrInternalError, "SMTP settings not configured")
		}

		emailFile, err := resolveEmailAttachment(files, smtpConfig.MaxAttachmentMB)
		if err != nil {
			return err
		}
		filePath = emailFile.Path

		m := mailer.NewSMTPMailer(smtpConfig)

		attachment := &mailer.Attachment{
			Filename: filepath.Base(filePath),
			Path:     filePath,
		}

		authorName := ""
		if book.AuthorName != nil {
			authorName = *book.AuthorName
		}

		subject := fmt.Sprintf("Book Delivery: %s", book.Title)
		body := fmt.Sprintf("Here is your requested book: %s by %s.", book.Title, authorName)

		if err := m.SendEmail(device.TargetAddress, subject, body, attachment); err != nil {
			return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to send email to device: %v", err))
		}

	case "koreader":
		client := netx.NewSafeHTTPClient(45 * time.Second)

		file, err := os.Open(filePath)
		if err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to open book file")
		}
		defer file.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to create multipart form")
		}
		if _, err := io.Copy(part, file); err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to stream file data")
		}
		_ = writer.Close()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, device.TargetAddress, body)
		if err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to construct HTTP request")
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := client.Do(req)
		if err != nil {
			return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to push file to KOReader: %v", err))
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("KOReader endpoint returned HTTP %d", resp.StatusCode))
		}
	}

	return nil
}
