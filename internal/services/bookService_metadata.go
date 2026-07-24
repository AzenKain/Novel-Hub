package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

func ensureAuthor(ctx context.Context, repo repositories.BookDBRepository, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	author, err := repo.GetAuthorByName(ctx, name)
	if err == nil && author != nil {
		return author.ID, nil
	}
	newAuthorID := uuid.Must(uuid.NewV7()).String()
	if err := repo.CreateAuthor(ctx, &models.AuthorEntity{
		ID:   newAuthorID,
		Name: name,
	}); err != nil {
		return "", err
	}
	return newAuthorID, nil
}

func mergeBookMetadataJSON(existing *string, req *request.UpdateBookMetadataDto) (string, error) {
	metaMap := map[string]any{}
	if existing != nil && strings.TrimSpace(*existing) != "" {
		_ = jsonx.UnmarshalString(*existing, &metaMap)
	}
	setStringMetadata(metaMap, "title", req.Title)
	setStringMetadata(metaMap, "creator", req.Author)
	setStringMetadata(metaMap, "description", req.Description)
	setStringMetadata(metaMap, "publisher", req.Publisher)
	setStringMetadata(metaMap, "language", req.Language)
	setStringMetadata(metaMap, "date", req.Date)
	setStringMetadata(metaMap, "series", req.Series)
	setStringMetadata(metaMap, "seriesIndex", req.SeriesIndex)
	if len(req.Subjects) > 0 {
		metaMap["subject"] = req.Subjects
	} else {
		delete(metaMap, "subject")
	}

	rawMeta, _ := metaMap["meta"].([]any)
	rawMeta = upsertMetaValue(rawMeta, "calibre:series", req.Series)
	rawMeta = upsertMetaValue(rawMeta, "calibre:series_index", req.SeriesIndex)
	if len(rawMeta) > 0 {
		metaMap["meta"] = rawMeta
	} else {
		delete(metaMap, "meta")
	}
	return jsonx.MarshalString(metaMap)
}

func setStringMetadata(metaMap map[string]any, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		metaMap[key] = value
	} else {
		delete(metaMap, key)
	}
}

func upsertMetaValue(rawMeta []any, name string, value string) []any {
	value = strings.TrimSpace(value)
	var updated []any
	found := false
	for _, item := range rawMeta {
		m, ok := item.(map[string]any)
		if !ok {
			updated = append(updated, item)
			continue
		}
		if m["name"] == name || m["property"] == name {
			found = true
			if value != "" {
				m["content"] = value
				updated = append(updated, m)
			}
			continue
		}
		updated = append(updated, item)
	}
	if !found && value != "" {
		updated = append(updated, map[string]any{
			"name":    name,
			"content": value,
		})
	}
	return updated
}

func fallbackCoverFromImages(parser bookparser.Parser, filePath, format string) ([]byte, string, error) {
	images, err := parser.ListImages(filePath)
	if err != nil {
		return nil, "", err
	}
	for _, imagePath := range images {
		data, err := parser.GetAsset(filePath, imagePath)
		if err != nil || len(data) == 0 {
			continue
		}
		if len(data) > constants.HardMaxCoverBytes || (strings.EqualFold(format, "pdf") && !bookparser.IsSuitablePDFCover(data)) || bookparser.IsBlankImage(data) {
			continue
		}
		contentType := readerAssetContentType(imagePath)
		if !isSupportedCoverContentType(contentType) {
			contentType = http.DetectContentType(data)
		}
		if isSupportedCoverContentType(contentType) {
			return data, contentType, nil
		}
	}
	return nil, "", fmt.Errorf("no supported image cover found")
}
