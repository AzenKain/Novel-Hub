package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

type SystemOperationsController struct {
	logs    services.SystemLogService
	backups services.BackupService
	audit   services.AuditService
}

func NewSystemOperationsController(logs services.SystemLogService, backups services.BackupService, audit services.AuditService) *SystemOperationsController {
	return &SystemOperationsController{logs: logs, backups: backups, audit: audit}
}

func systemOperationContext(c fiber.Ctx) (context.Context, context.CancelFunc) {
	return auditContext(c, 30*time.Minute)
}

func (c *SystemOperationsController) ListLogs(ctx fiber.Ctx) error {
	reqCtx, cancel := systemOperationContext(ctx)
	defer cancel()

	items, err := c.logs.List(reqCtx)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: items})
}

func (c *SystemOperationsController) TailLogs(ctx fiber.Ctx) error {
	reqCtx, cancel := systemOperationContext(ctx)
	defer cancel()

	dto := &request.LogTailDto{Lines: 200}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	result, err := c.logs.Tail(reqCtx, dto.File, dto.Lines, dto.Level, dto.Search)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: result})
}

func (c *SystemOperationsController) DownloadLog(ctx fiber.Ctx) error {
	_, cancel := systemOperationContext(ctx)
	defer cancel()

	path, err := c.logs.Path(ctx.Params("name"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.Download(path, ctx.Params("name"))
}

func (c *SystemOperationsController) ListBackups(ctx fiber.Ctx) error {
	reqCtx, cancel := systemOperationContext(ctx)
	defer cancel()

	items, err := c.backups.List(reqCtx)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: items})
}

func (c *SystemOperationsController) CreateBackup(ctx fiber.Ctx) error {
	reqCtx, cancel := systemOperationContext(ctx)
	defer cancel()

	dto := &request.CreateBackupDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	backup, err := c.backups.Create(reqCtx, dto.IncludeBooks)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	c.audit.Record(reqCtx, services.AuditActionBackupCreate, "backup", backup.Name, backup.Name)
	return ctx.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: backup})
}

func (c *SystemOperationsController) DownloadBackup(ctx fiber.Ctx) error {
	_, cancel := systemOperationContext(ctx)
	defer cancel()

	path, err := c.backups.Path(ctx.Params("name"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.Download(path, ctx.Params("name"))
}

func (c *SystemOperationsController) DeleteBackup(ctx fiber.Ctx) error {
	reqCtx, cancel := systemOperationContext(ctx)
	defer cancel()

	if err := c.backups.Delete(reqCtx, ctx.Params("name")); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	c.audit.Record(reqCtx, services.AuditActionBackupDelete, "backup", ctx.Params("name"), ctx.Params("name"))
	return ctx.JSON(response.CommonResponse{Status: true})
}

func (c *SystemOperationsController) RestoreBackup(ctx fiber.Ctx) error {
	reqCtx, cancel := systemOperationContext(ctx)
	defer cancel()

	dto := &request.RestoreBackupDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	result, err := c.backups.StageRestore(reqCtx, ctx.Params("name"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	c.audit.Record(reqCtx, services.AuditActionBackupRestore, "backup", ctx.Params("name"), ctx.Params("name"))
	return ctx.Status(fiber.StatusAccepted).JSON(response.CommonResponse{Status: true, Data: result})
}
