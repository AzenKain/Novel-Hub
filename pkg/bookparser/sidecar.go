package bookparser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func MetadataSidecarPath(filePath string) string {
	return filePath + ".novelhub-metadata.json"
}

func SaveMetadataSidecar(filePath string, meta *BookMetadata) error {
	if meta == nil {
		return fmt.Errorf("metadata is nil")
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata sidecar: %w", err)
	}
	if err := os.WriteFile(MetadataSidecarPath(filePath), data, 0600); err != nil {
		return fmt.Errorf("write metadata sidecar: %w", err)
	}
	return nil
}

func MergeMetadataSidecar(filePath string, meta *BookMetadata) *BookMetadata {
	if meta == nil {
		meta = &BookMetadata{}
	}
	data, err := os.ReadFile(MetadataSidecarPath(filePath))
	if err != nil {
		return meta
	}
	var sidecar BookMetadata
	if err := json.Unmarshal(data, &sidecar); err != nil {
		return meta
	}
	overlayMetadata(meta, &sidecar)
	return meta
}

func overlayMetadata(target *BookMetadata, source *BookMetadata) {
	if strings.TrimSpace(source.Title) != "" {
		target.Title = source.Title
	}
	if strings.TrimSpace(source.Author) != "" {
		target.Author = source.Author
	}
	if strings.TrimSpace(source.Description) != "" {
		target.Description = source.Description
	}
	if strings.TrimSpace(source.Publisher) != "" {
		target.Publisher = source.Publisher
	}
	if strings.TrimSpace(source.Language) != "" {
		target.Language = source.Language
	}
	if strings.TrimSpace(source.Date) != "" {
		target.Date = source.Date
	}
	if len(source.Subjects) > 0 {
		target.Subjects = append([]string(nil), source.Subjects...)
	}
	if strings.TrimSpace(source.Series) != "" {
		target.Series = source.Series
	}
	if strings.TrimSpace(source.SeriesIndex) != "" {
		target.SeriesIndex = source.SeriesIndex
	}
	if len(source.CoverData) > 0 {
		target.CoverData = append([]byte(nil), source.CoverData...)
		target.CoverType = source.CoverType
	}
	if strings.TrimSpace(source.MetadataJSON) != "" {
		target.MetadataJSON = source.MetadataJSON
	}
}
