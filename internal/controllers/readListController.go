package controllers

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/validator"
)

type ReadListController struct {
	service services.ReadListService
}

func NewReadListController(service services.ReadListService) *ReadListController {
	return &ReadListController{service: service}
}

func (c *ReadListController) GetReadLists(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}

	dto := &request.GetReadListsDto{}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	cursorCreatedAt, cursorID := dto.ParseCursor()
	limit := dto.GetLimit()

	lists, err := c.service.GetUserReadLists(reqCtx, userID, cursorCreatedAt, cursorID, limit)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	var nextCursor *string
	if len(lists) > 0 && len(lists) >= int(limit) {
		last := lists[len(lists)-1]
		next := last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID
		nextCursor = &next
	}
	return ctx.JSON(fiber.Map{"status": true, "data": lists, "next_cursor": nextCursor})
}

func (c *ReadListController) GetReadList(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	list, err := c.service.GetReadList(reqCtx, ctx.Params("id"), userID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: list})
}

func (c *ReadListController) CreateReadList(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	dto := &request.CreateReadListDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	list, err := c.service.CreateReadList(reqCtx, userID, *dto)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: list})
}

func (c *ReadListController) UpdateReadList(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	dto := &request.UpdateReadListDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	list, err := c.service.UpdateReadList(reqCtx, ctx.Params("id"), userID, *dto)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: list})
}

func (c *ReadListController) DeleteReadList(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	if err := c.service.DeleteReadList(reqCtx, ctx.Params("id"), userID); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Message: "Read list deleted"})
}

func (c *ReadListController) GetReadListBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	claims, _ := getUserClaims(ctx)
	books, err := c.service.GetReadListBooks(reqCtx, ctx.Params("id"), userID, claims)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: books})
}

func (c *ReadListController) AddBook(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	dto := &request.AddReadListBookDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, _ := getUserClaims(ctx)
	if err := c.service.AddBook(reqCtx, ctx.Params("id"), userID, dto.BookID, claims); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Message: "Book added to read list"})
}

func (c *ReadListController) RemoveBook(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	if err := c.service.RemoveBook(reqCtx, ctx.Params("id"), userID, ctx.Params("bookId")); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Message: "Book removed from read list"})
}

func (c *ReadListController) Reorder(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	dto := &request.ReorderReadListDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	if err := c.service.Reorder(reqCtx, ctx.Params("id"), userID, dto.BookIDs); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Message: "Read list reordered"})
}

func (c *ReadListController) NextInOrder(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	dto := &request.NextInOrderQueryDto{}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, _ := getUserClaims(ctx)
	next, err := c.service.NextInOrder(reqCtx, ctx.Params("id"), userID, dto.After, claims)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: next})
}


func (c *ReadListController) ImportCBL(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "unauthorized"})
	}
	header, err := ctx.FormFile("file")
	if err != nil || header == nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "A .cbl file is required"})
	}
	if header.Size > constants.MaxCBLBytes {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "The .cbl file is too large"})
	}
	file, err := header.Open()
	if err != nil {
		return apperrors.HandleError(ctx, apperrors.New(apperrors.ErrInternalError, "Failed to open the uploaded file"))
	}
	defer file.Close()

	name := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	result, err := c.service.ImportCBL(reqCtx, userID, io.Reader(file), name)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: result})
}
