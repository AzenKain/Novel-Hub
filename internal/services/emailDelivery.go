package services

import (
	"fmt"
	"os"
	"strings"

	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
)

var emailDeliverableFormats = map[string]int{
	"EPUB": 0,
	"PDF":  1,
	"DOCX": 2,
	"DOC":  3,
	"RTF":  4,
	"HTML": 5,
	"HTM":  6,
	"TXT":  7,
}

func pickEmailDeliverableFile(files []*models.BookFileEntity) *models.BookFileEntity {
	var best *models.BookFileEntity
	bestRank := len(emailDeliverableFormats)
	for _, file := range files {
		if file == nil {
			continue
		}
		rank, ok := emailDeliverableFormats[strings.ToUpper(file.Format)]
		if ok && rank < bestRank {
			best, bestRank = file, rank
		}
	}
	return best
}

func resolveEmailAttachment(files []*models.BookFileEntity, maxAttachmentMB int) (*models.BookFileEntity, error) {
	target := pickEmailDeliverableFile(files)
	if target == nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "This book has no e-reader compatible file (EPUB, PDF, DOCX, RTF, HTML or TXT)")
	}
	info, err := os.Stat(target.Path)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Book file missing from disk")
	}
	if maxAttachmentMB <= 0 {
		maxAttachmentMB = 50
	}
	if info.Size() > int64(maxAttachmentMB)*1024*1024 {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("Book file exceeds %dMB email attachment size limit", maxAttachmentMB))
	}
	return target, nil
}
