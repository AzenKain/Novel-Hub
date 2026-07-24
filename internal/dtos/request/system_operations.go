package request

type CreateBackupDto struct {
	IncludeBooks bool `json:"include_books"`
}

type RestoreBackupDto struct {
	Confirmation string `json:"confirmation" validate:"required,eq=RESTORE"`
}

type LogTailDto struct {
	File   string `query:"file" validate:"required,max=100"`
	Lines  int    `query:"lines" validate:"omitempty,min=1,max=2000"`
	Level  string `query:"level" validate:"omitempty,oneof=trace debug info warn error fatal panic"`
	Search string `query:"search" validate:"omitempty,max=200"`
}
