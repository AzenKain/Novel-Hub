package request

type CalibrePaginationDto struct {
	Num       int64  `query:"num" validate:"omitempty,min=1,max=1000"`
	Offset    int64  `query:"offset" validate:"omitempty,min=0"`
	Sort      string `query:"sort" validate:"omitempty,max=50"`
	SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc ASC DESC"`
}

type CalibreSearchQueryDto struct {
	Query     string `query:"query" validate:"omitempty,max=500"`
	Num       int64  `query:"num" validate:"omitempty,min=1,max=1000"`
	Offset    int64  `query:"offset" validate:"omitempty,min=0"`
	Sort      string `query:"sort" validate:"omitempty,max=50"`
	SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc ASC DESC"`
}

type CalibreBooksQueryDto struct {
	IDs          string `query:"ids"`
	CategoryUrls string `query:"category_urls"`
	IDIsUUID     string `query:"id_is_uuid"`
}
