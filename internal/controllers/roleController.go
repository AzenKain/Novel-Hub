package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

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

	res, ferr := h.service.GetRoleByID(ctx, c.Params("id"))
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) GetAllRole(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, ferr := h.service.GetAllRole(ctx)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) GetPermissions(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, ferr := h.service.GetPermissions(ctx)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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
	res, ferr := h.service.CreateRole(ctx, dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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
	res, ferr := h.service.UpdateRole(ctx, c.Params("id"), dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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
	res, ferr := h.service.UpdateRolePermissions(ctx, c.Params("id"), dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *RoleController) DeleteRole(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ferr := h.service.DeleteRole(ctx, c.Params("id"))
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "Role deleted successfully"})
}
