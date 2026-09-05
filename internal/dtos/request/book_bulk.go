package request

type BulkDeleteBooksDto struct {
	BookIDs []string `json:"book_ids" validate:"required,min=1,max=100,dive,required"`
}

type BulkMoveBooksDto struct {
	BookIDs         []string `json:"book_ids" validate:"required,min=1,max=100,dive,required"`
	TargetLibraryID string   `json:"target_library_id" validate:"required"`
}

type BulkAssignCollectionsDto struct {
	BookIDs       []string `json:"book_ids" validate:"required,min=1,max=100,dive,required"`
	CollectionIDs []string `json:"collection_ids" validate:"required,min=1,max=20,dive,required"`
}

type BulkAddTagsDto struct {
	BookIDs  []string `json:"book_ids" validate:"required,min=1,max=100,dive,required"`
	TagNames []string `json:"tag_names" validate:"required,min=1,max=20,dive,required"`
}

type BulkUpdateMetadataItemDto struct {
	BookID      string  `json:"book_id" validate:"required"`
	Title       *string `json:"title,omitempty" validate:"omitempty"`
	Author      *string `json:"author,omitempty" validate:"omitempty"`
	SeriesIndex *string `json:"series_index,omitempty" validate:"omitempty"`
	Publisher   *string `json:"publisher,omitempty" validate:"omitempty"`
	Language    *string `json:"language,omitempty" validate:"omitempty"`
	Description *string `json:"description,omitempty" validate:"omitempty"`
}

type BulkUpdateMetadataDto struct {
	BookIDs         []string                    `json:"book_ids" validate:"required,min=1,max=100,dive,required"`
	Author          *string                     `json:"author,omitempty" validate:"omitempty"`
	Series          *string                     `json:"series,omitempty" validate:"omitempty"`
	AutoIndexSeries bool                        `json:"auto_index_series,omitempty"`
	Publisher       *string                     `json:"publisher,omitempty" validate:"omitempty"`
	Language        *string                     `json:"language,omitempty" validate:"omitempty"`
	AddTags         []string                    `json:"add_tags,omitempty" validate:"omitempty,max=50,dive,required"`
	RemoveTags      []string                    `json:"remove_tags,omitempty" validate:"omitempty,max=50,dive,required"`
	Items           []BulkUpdateMetadataItemDto `json:"items,omitempty" validate:"omitempty,dive"`
}
