package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
)

type JobController struct {
	service services.JobService
}

func NewJobController(service services.JobService) *JobController {
	return &JobController{service: service}
}

func (c *JobController) GetJob(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jobID := ctx.Params("id")
	job, err := c.service.GetJob(reqCtx, jobID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Job not found"})
	}

	return ctx.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: job})
}
