package bookparser

const RawFileContentPath = "__novelhub_raw_file__"

type BookMetadata struct {
	Title        string
	Author       string
	Description  string
	Publisher    string
	Language     string
	Date         string
	Subjects     []string
	Series       string
	SeriesIndex  string
	CoverData    []byte
	CoverType    string
	MetadataJSON string
	ReadingDirection string
}

type ChapterData struct {
	Title       string
	Content     string
	ContentPath string
	Index       int
}

type BookData struct {
	Metadata BookMetadata
	Chapters []ChapterData
}

type Parser interface {
	ParseMetadata(filePath string) (*BookMetadata, error)

	ParseSpine(filePath string) ([]ChapterData, error)

	ParseBook(filePath string) (*BookData, error)

	GetChapterContent(filePath, contentPath string) (string, error)

	GetAsset(filePath, assetPath string) ([]byte, error)

	ListImages(filePath string) ([]string, error)

	SaveOriginalMetadataAndFix(filePath string, meta *BookMetadata) error
}
