package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/validator"
	"novelhub/pkg/apperrors"
)

type UserController struct {
	service services.UserService
}

func NewUserController(svc services.UserService) *UserController {
	return &UserController{service: svc}
}

func (h *UserController) CreateUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.CreateUserDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.CreateUser(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) GetUserCurrent(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uid, ok := c.Locals("uid").(string)
	if !ok || uid == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.GetUserCurrent(ctx, uid)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) UpdateProfile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.UpdateProfileDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.UpdateProfile(ctx, claims.UId, claims, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) ChangePassword(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.ChangePasswordDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	uid, ok := c.Locals("uid").(string)
	if !ok || uid == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	err := h.service.ChangePassword(ctx, uid, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "Password changed successfully"})
}

func (h *UserController) SearchUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchUserDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.SearchUser(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

func (h *UserController) GetUserByID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.GetUserByID(ctx, c.Params("id"))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) AdminUpdateProfile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.UpdateProfileDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.UpdateProfile(ctx, c.Params("id"), claims, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) AdminResetPassword(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.ResetPasswordDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	err := h.service.AdminResetPassword(ctx, c.Params("id"), claims, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "Password reset successfully"})
}

func (h *UserController) ChangeRoleUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.ChangeRoleDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.ChangeRoleUser(ctx, c.Params("id"), claims, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) RestoreUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.RestoreUser(ctx, c.Params("id"))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) DeleteUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if ok && claims.UId == c.Params("id") {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You cannot delete yourself"})
	}

	err := h.service.DeleteUser(ctx, c.Params("id"), claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "User deleted successfully"})
}
