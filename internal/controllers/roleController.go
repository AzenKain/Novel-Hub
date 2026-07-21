package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"novelhub/pkg/apperrors"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/validator"
)

type RoleController struct {
	service services.RoleService
}

func NewRoleController(svc services.RoleService) *RoleController {
	return &RoleController{service: svc}
}

func (h *RoleController) GetRoleByID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.GetRoleByID(ctx, c.Params("id"))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) GetAllRole(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.GetAllRole(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) GetPermissions(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.GetPermissions(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) CreateRole(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.CreateRoleDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	res, err := h.service.CreateRole(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) UpdateRole(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.UpdateRoleDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	res, err := h.service.UpdateRole(ctx, c.Params("id"), dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) UpdateRolePermissions(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.UpdateRolePermissionsDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	res, err := h.service.UpdateRolePermissions(ctx, c.Params("id"), dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) DeleteRole(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := h.service.DeleteRole(ctx, c.Params("id"))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "Role deleted successfully"})
}
