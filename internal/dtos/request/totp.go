package request

type TOTPCodeDto struct {
	Code string `json:"code" validate:"required,min=6,max=16"`
}
