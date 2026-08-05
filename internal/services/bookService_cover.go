package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
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

	coverURLPath, _, err := s.fileRepo.SaveCover(ctx, bookID, ext, coverData)
	if err != nil {
		return "", err
	}

	if err := s.updateCoverURL(ctx, bookID, coverURLPath); err != nil {
		return "", err
	}
	return coverURLPath, nil
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
