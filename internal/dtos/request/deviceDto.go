package request

type CreateUserDeviceDto struct {
	Name          string `json:"name" validate:"required,min=1,max=100"`
	DeviceType    string `json:"device_type" validate:"required,oneof=kindle pocketbook koreader"`
	TargetAddress string `json:"target_address" validate:"required,min=1,max=255"`
}

type PushBookToDeviceDto struct {
	DeviceID string `json:"device_id" validate:"required"`
}

type ListUserDevicesQueryDto struct {
	Cursor string `query:"cursor" validate:"omitempty,max=255"`
	Limit  int64  `query:"limit" validate:"omitempty,min=1,max=100"`
}
