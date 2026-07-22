package controllers

import (
	"github.com/gofiber/fiber/v3"
	"novelhub/internal/dtos/response"
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
