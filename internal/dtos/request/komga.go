package request

type KomgaListSeriesQueryDto struct {
	Page    int64  `query:"page" validate:"omitempty,min=0"`
	Size    int64  `query:"size" validate:"omitempty,min=0,max=100"`
	Unpaged bool   `query:"unpaged"`
	Search  string `query:"search" validate:"omitempty,max=255"`
}

type KomgaListBooksQueryDto struct {
	Page int64 `query:"page" validate:"omitempty,min=0"`
	Size int64 `query:"size" validate:"omitempty,min=0,max=100"`
}
