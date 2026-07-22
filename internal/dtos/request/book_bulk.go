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
