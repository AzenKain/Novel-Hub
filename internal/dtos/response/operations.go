package response

import "time"

type JobTaskResponse struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type JobScheduleResponse struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	TaskType        string     `json:"taskType"`
	PayloadJSON     *string    `json:"payloadJson,omitempty"`
	IntervalMinutes int64      `json:"intervalMinutes"`
	Enabled         bool       `json:"enabled"`
	NextRunAt       time.Time  `json:"nextRunAt"`
	LastRunAt       *time.Time `json:"lastRunAt,omitempty"`
	LastJobID       *string    `json:"lastJobId,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}
