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

	res, ferr := h.service.CreateUser(ctx, dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) GetUserCurrent(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, ferr := h.service.GetUserCurrent(ctx, c.Locals("uid").(string))
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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

	res, ferr := h.service.UpdateProfile(ctx, c.Locals("uid").(string), dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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

	ferr := h.service.ChangePassword(ctx, c.Locals("uid").(string), dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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

	res, ferr := h.service.SearchUser(ctx, dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

func (h *UserController) GetUserByID(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, ferr := h.service.GetUserByID(ctx, c.Params("id"))
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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

	res, ferr := h.service.UpdateProfile(ctx, c.Params("id"), dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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

	ferr := h.service.AdminResetPassword(ctx, c.Params("id"), dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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

	claims, ok := c.Locals("user_claims").(*response.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, ferr := h.service.ChangeRoleUser(ctx, c.Params("id"), claims, dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) RestoreUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, ferr := h.service.RestoreUser(ctx, c.Params("id"))
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *UserController) DeleteUser(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := c.Locals("user_claims").(*response.JWTClaims)
	if ok && claims.UId == c.Params("id") {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You cannot delete yourself"})
	}

	ferr := h.service.DeleteUser(ctx, c.Params("id"))
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "User deleted successfully"})
}
