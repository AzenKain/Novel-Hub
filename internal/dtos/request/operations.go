package request

type ListJobsDto struct {
	Status string `query:"status" validate:"omitempty,oneof=pending running completed failed"`
	Type   string `query:"type" validate:"omitempty,max=64"`
	Limit  int64  `query:"limit" validate:"omitempty,min=1,max=100"`
	Offset int64  `query:"offset" validate:"omitempty,min=0"`
}

type TriggerJobDto struct {
	Type        string `json:"type" validate:"required,max=64"`
	PayloadJSON string `json:"payload_json" validate:"omitempty,max=4096"`
}

type UpsertJobScheduleDto struct {
	Name            string `json:"name" validate:"required,min=1,max=120"`
	TaskType        string `json:"task_type" validate:"required,max=64"`
	PayloadJSON     string `json:"payload_json" validate:"omitempty,max=4096"`
	IntervalMinutes int64  `json:"interval_minutes" validate:"required,min=5,max=525600"`
	Enabled         bool   `json:"enabled"`
}
