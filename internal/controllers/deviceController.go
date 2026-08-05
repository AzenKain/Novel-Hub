package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/convert"
	"novelhub/pkg/validator"
)

type DeviceController struct {
	service services.DeviceService
}

func NewDeviceController(svc services.DeviceService) *DeviceController {
	return &DeviceController{service: svc}
}

func (ctrl *DeviceController) ListDevices(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}

	queryDto := &request.ListUserDevicesQueryDto{Limit: 20}
	if err := validator.ValidateQueryDto(c, queryDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var cursor *time.Time
	var cursorID string
	if queryDto.Cursor != "" {
		if parts := convert.DecodeCursor(queryDto.Cursor); len(parts) == 2 {
			if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursor = &parsedTime
				cursorID = parts[1]
			}
		}
	}

	devices, err := ctrl.service.ListDevices(ctx, userID, cursor, cursorID, queryDto.Limit)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: devices})
}

func (ctrl *DeviceController) CreateDevice(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}

	dto := &request.CreateUserDeviceDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	device, err := ctrl.service.CreateDevice(ctx, userID, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: device})
}

func (ctrl *DeviceController) DeleteDevice(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}

	deviceID := c.Params("id")
	if deviceID == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Device ID required"))
	}

	if err := ctrl.service.DeleteDevice(ctx, deviceID, userID); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Device removed successfully"})
}

func (ctrl *DeviceController) PushBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}

	bookID := c.Params("id")
	if bookID == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Book ID required"))
	}

	dto := &request.PushBookToDeviceDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims := getOptionalClaims(c)
	if err := ctrl.service.PushBookToDevice(ctx, userID, bookID, dto.DeviceID, claims); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Book delivered to device successfully"})
}
