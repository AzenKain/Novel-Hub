package models

type ReaderBootstrapEntity struct {
	Book     *BookEntity      `json:"book"`
	Chapters []*ChapterEntity `json:"chapters"`
}

type ReaderAssetEntity struct {
	Data        []byte `json:"-"`
	ContentType string `json:"contentType"`
}
