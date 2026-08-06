package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/jsonx"
)

type KomgaController struct {
	komgaService services.KomgaService
}

func NewKomgaController(komgaService services.KomgaService) *KomgaController {
	return &KomgaController{komgaService: komgaService}
}

func komgaContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func (ctrl *KomgaController) claims(c fiber.Ctx) (*response.JWTClaims, error) {
	claims, ok := getUserClaims(c)
	if !ok {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}
	return claims, nil
}

func (ctrl *KomgaController) ListSeries(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	page, _ := strconv.ParseInt(c.Query("page", "0"), 10, 64)
	size, _ := strconv.ParseInt(c.Query("size", "20"), 10, 64)
	// unpaged=true asks for everything in one response; the extension uses it for chapter lists.
	if c.Query("unpaged") == "true" {
		size = 100
	}

	res, err := ctrl.komgaService.ListSeries(ctx, c.Query("search"), page, size, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KomgaController) GetSeries(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	res, err := ctrl.komgaService.GetSeries(ctx, c.Params("seriesId"), claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KomgaController) ListSeriesBooks(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	books, err := ctrl.komgaService.ListSeriesBooks(ctx, c.Params("seriesId"), claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	count := int64(len(books))
	return c.JSON(response.KomgaPageWrapper[response.KomgaBook]{
		Content:          books,
		Empty:            count == 0,
		First:            true,
		Last:             true,
		Number:           0,
		NumberOfElements: count,
		Size:             count,
		TotalElements:    count,
		TotalPages:       1,
	})
}

func (ctrl *KomgaController) GetBook(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	res, err := ctrl.komgaService.GetBook(ctx, c.Params("bookId"), claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KomgaController) ListBookPages(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	pages, err := ctrl.komgaService.ListBookPages(ctx, c.Params("bookId"), claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(pages)
}

func (ctrl *KomgaController) GetBookPage(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	pageNumber, convErr := strconv.Atoi(c.Params("pageNumber"))
	if convErr != nil {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Invalid page number"))
	}
	asset, err := ctrl.komgaService.GetBookPage(ctx, c.Params("bookId"), pageNumber, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	c.Set(fiber.HeaderContentType, asset.ContentType)
	c.Set("Cache-Control", "private, max-age=3600")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(asset.Data)
}

func (ctrl *KomgaController) ListLibraries(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	libs, err := ctrl.komgaService.ListLibraries(ctx, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(libs)
}

func (ctrl *KomgaController) GetSeriesProgress(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	res, err := ctrl.komgaService.SeriesProgress(ctx, c.Params("seriesId"), claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KomgaController) UpdateSeriesProgress(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	var payload response.KomgaReadProgressUpdateV2
	if err := jsonx.Unmarshal(c.Body(), &payload); err != nil {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Invalid read progress payload"))
	}
	if err := ctrl.komgaService.MarkSeriesReadUpTo(ctx, c.Params("seriesId"), payload.LastBookNumberSortRead, claims); err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (ctrl *KomgaController) GetSeriesThumbnail(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	path, err := ctrl.komgaService.SeriesCoverPath(ctx, c.Params("seriesId"), claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.SendFile(path)
}

func (ctrl *KomgaController) GetBookThumbnail(c fiber.Ctx) error {
	ctx, cancel := komgaContext()
	defer cancel()

	claims, err := ctrl.claims(c)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	path, err := ctrl.komgaService.BookCoverPath(ctx, c.Params("bookId"), claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.SendFile(path)
}
