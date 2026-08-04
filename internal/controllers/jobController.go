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

type JobController struct {
	service  services.JobService
	schedule services.JobScheduleService
}

func NewJobController(service services.JobService, schedule services.JobScheduleService) *JobController {
	return &JobController{service: service, schedule: schedule}
}

func (c *JobController) GetJob(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	job, err := c.service.GetJob(reqCtx, ctx.Params("id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: job})
}

func (c *JobController) ListJobs(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dto := &request.ListJobsDto{}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	if dto.Limit <= 0 || dto.Limit > 100 {
		dto.Limit = 50
	}

	jobs, total, err := c.service.ListJobs(reqCtx, dto.Status, dto.Type, dto.Limit, dto.Offset)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	page := int(dto.Offset/dto.Limit) + 1
	return ctx.JSON(response.BuildPaginatedResponse(jobs, total, page, int(dto.Limit)))
}

func (c *JobController) ListTasks(ctx fiber.Ctx) error {
	data := c.service.ListTasks()
	return ctx.JSON(response.CommonResponse{Status: true, Data: data})
}

func (c *JobController) TriggerJob(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dto := &request.TriggerJobDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	job, err := c.service.Trigger(reqCtx, dto.Type, dto.PayloadJSON)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.Status(fiber.StatusAccepted).JSON(response.CommonResponse{Status: true, Data: job})
}

func (c *JobController) ListSchedules(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	items, err := c.schedule.List(reqCtx)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: items})
}

func (c *JobController) CreateSchedule(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dto := &request.UpsertJobScheduleDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	item, err := c.schedule.Create(reqCtx, dto)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: item})
}

func (c *JobController) UpdateSchedule(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dto := &request.UpsertJobScheduleDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	item, err := c.schedule.Update(reqCtx, ctx.Params("id"), dto)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: item})
}

func (c *JobController) DeleteSchedule(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := c.schedule.Delete(reqCtx, ctx.Params("id")); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true})
}

func (c *JobController) RunSchedule(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	job, err := c.schedule.RunNow(reqCtx, ctx.Params("id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.Status(fiber.StatusAccepted).JSON(response.CommonResponse{Status: true, Data: job})
}
