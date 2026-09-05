package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/netx"
)

func (s *bookService) UpdateCover(ctx context.Context, bookID string, input request.UpdateCoverDto) (string, error) {
	if _, err := s.GetBook(ctx, bookID); err != nil {
		return "", err
	}

	coverData, ext, err := s.resolveCoverData(ctx, bookID, input)
	if err != nil {
		return "", err
	}

	ext, coverData = s.optimizeCoverIfEnabled(ctx, ext, coverData)

	coverURLPath, _, err := s.fileRepo.SaveCover(ctx, bookID, ext, coverData)
	if err != nil {
		return "", err
	}

	if err := s.updateCoverURL(ctx, bookID, coverURLPath); err != nil {
		return "", err
	}
	return coverURLPath, nil
}

func (s *bookService) GetBookCoverPath(ctx context.Context, bookID string, claims *response.JWTClaims) (string, error) {
	book, err := s.GetBook(ctx, bookID)
	if err != nil {
		return "", err
	}

	if !s.CanReadBook(ctx, book, claims) {
		return "", apperrors.New(apperrors.ErrForbidden, "Access denied")
	}

	if book.CoverURL == nil || strings.TrimSpace(*book.CoverURL) == "" {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}

	resolved, err := s.fileRepo.ResolveCoverPath(ctx, book.ID, *book.CoverURL)
	if err != nil {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}

	if _, err := os.Stat(resolved); err != nil {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover file missing on disk")
	}

	return resolved, nil
}

func (s *bookService) resolveCoverData(ctx context.Context, bookID string, input request.UpdateCoverDto) ([]byte, string, error) {
	limit := s.settings.Limits().CoverBytes
	if len(input.UploadedData) > 0 {
		ext, err := bookparser.ValidateImage(input.UploadedData, limit)
		if err != nil {
			return nil, "", err
		}
		return input.UploadedData, ext, nil
	}

	if input.EPUBImagePath != "" {
		file, err := s.GetBookFile(ctx, bookID, "")
		if err != nil {
			return nil, "", err
		}
		parser, err := s.parserForFile(file)
		if err != nil {
			return nil, "", err
		}
		coverData, err := parser.GetAsset(file.Path, input.EPUBImagePath)
		if err != nil {
			return nil, "", err
		}
		ext, err := bookparser.ValidateImage(coverData, limit)
		if err != nil {
			return nil, "", err
		}
		return coverData, ext, nil
	}

	if input.CoverURL != "" {
		parsed, err := url.Parse(input.CoverURL)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cover URL")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, "", fmt.Errorf("cover URL must use http or https scheme")
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.CoverURL, nil)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36 Edg/151.0.0.0")
		req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		if parsed.Host != "" {
			req.Header.Set("Referer", parsed.Scheme+"://"+parsed.Host+"/")
		}
		client := netx.NewSafeHTTPClient(15 * time.Second)
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("cover download blocked or failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, "", fmt.Errorf("cover download failed with status %d", resp.StatusCode)
		}
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		if ct != "" && !strings.HasPrefix(ct, "image/") {
			return nil, "", fmt.Errorf("cover URL did not return an image (got %s)", ct)
		}
		coverData, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		if err != nil {
			return nil, "", err
		}
		ext, err := bookparser.ValidateImage(coverData, limit)
		if err != nil {
			return nil, "", err
		}
		return coverData, ext, nil
	}

	return nil, "", fmt.Errorf("no cover provided")
}

func (s *bookService) updateCoverURL(ctx context.Context, bookID string, coverURL string) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return err
	}
	book.CoverURL = &coverURL
	if err := s.bookRepo.UpdateBook(ctx, book); err != nil {
		return err
	}
	return nil
}

func (s *bookService) ProxyCover(ctx context.Context, coverURL string) ([]byte, string, error) {
	parsed, err := url.Parse(coverURL)
	if err != nil {
		return nil, "", apperrors.New(apperrors.ErrBadRequest, "invalid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", apperrors.New(apperrors.ErrBadRequest, "URL must use http or https scheme")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, coverURL, nil)
	if err != nil {
		return nil, "", apperrors.New(apperrors.ErrInternalError, err.Error())
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	if parsed.Host != "" {
		req.Header.Set("Referer", parsed.Scheme+"://"+parsed.Host+"/")
	}

	client := netx.NewSafeHTTPClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", apperrors.New(apperrors.ErrInternalError, "failed to download image: "+err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", apperrors.New(apperrors.ErrInternalError, "failed to download image: status "+resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return nil, "", apperrors.New(apperrors.ErrBadRequest, "URL does not return a valid image type: "+contentType)
	}

	limit := s.settings.Limits().CoverBytes
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, "", apperrors.New(apperrors.ErrInternalError, "failed to read response: "+err.Error())
	}
	if int64(len(data)) > limit {
		return nil, "", apperrors.New(apperrors.ErrBadRequest, "image exceeds cover size limit")
	}

	ext, err := bookparser.ValidateImage(data, limit)
	if err != nil {
		return nil, "", apperrors.New(apperrors.ErrBadRequest, "invalid image content: "+err.Error())
	}

	var resolvedContentType string
	switch ext {
	case ".jpg":
		resolvedContentType = "image/jpeg"
	case ".png":
		resolvedContentType = "image/png"
	case ".gif":
		resolvedContentType = "image/gif"
	case ".webp":
		resolvedContentType = "image/webp"
	case ".bmp":
		resolvedContentType = "image/bmp"
	case ".tiff":
		resolvedContentType = "image/tiff"
	default:
		resolvedContentType = "application/octet-stream"
	}

	return data, resolvedContentType, nil
}

func (s *bookService) optimizeCoverIfEnabled(ctx context.Context, ext string, data []byte) (string, []byte) {
	if ext == ".svg" || len(data) == 0 {
		return ext, data
	}
	var settings *models.PublicSettings
	if s.settings != nil {
		if pub, err := s.settings.Public(ctx); err == nil {
			settings = pub
		}
	}
	if settings != nil && settings.EnableWebpCover {
		if webpData, ok, err := bookparser.ConvertToWebP(data); err == nil && ok {
			return ".webp", webpData
		}
	}
	return ext, data
}
