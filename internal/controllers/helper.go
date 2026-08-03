package controllers

import (
	"novelhub/internal/dtos/response"
	"novelhub/pkg/convert"

	"github.com/gofiber/fiber/v3"
)

func getOptionalClaims(c fiber.Ctx) *response.JWTClaims {
	claims, ok := c.Locals("user_claims").(*response.JWTClaims)
	if !ok {
		return nil
	}
	return claims
}

func getUserClaims(c fiber.Ctx) (*response.JWTClaims, bool) {
	claims, ok := c.Locals("user_claims").(*response.JWTClaims)
	if !ok || claims == nil {
		return nil, false
	}
	return claims, true
}

func getUserIdFromLocals(ctx fiber.Ctx) (string, bool) {
	uidRaw := ctx.Locals("uid")
	if uidRaw == nil {
		return "", false
	}
	uidStr, ok := uidRaw.(string)
	if !ok {
		return "", false
	}
	userID, err := convert.ParseID(uidStr)
	if err != nil {
		return "", false
	}
	return userID, true
}
