package request

type SendEmailDto struct {
	RecipientEmail string `json:"recipient_email" validate:"required,email"`
}
