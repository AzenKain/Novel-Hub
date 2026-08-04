package request

type CreateLibraryDto struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}

type UpdateLibraryDto struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}
