package models

import "novelhub/internal/dtos/response"

type ReaderBootstrapEntity struct {
	Book     *BookEntity      `json:"book"`
	Chapters []*ChapterEntity `json:"chapters"`
}

func (e *ReaderBootstrapEntity) ToResponse() *response.ReaderBootstrapResponse {
	if e == nil {
		return nil
	}
	var chs []*response.ChapterResponse
	if e.Chapters != nil {
		chs = ChapterEntitiesToResponse(e.Chapters)
	}
	return &response.ReaderBootstrapResponse{
		Book:     e.Book.ToResponse(),
		Chapters: chs,
	}
}

type ReaderAssetEntity struct {
	Data        []byte `json:"-"`
	ContentType string `json:"content_type"`
}

func (e *ReaderAssetEntity) ToResponse() *response.ReaderAssetResponse {
	if e == nil {
		return nil
	}
	return &response.ReaderAssetResponse{
		ContentType: e.ContentType,
	}
}
