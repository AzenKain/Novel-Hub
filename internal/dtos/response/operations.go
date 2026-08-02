package response

import "time"

type JobTaskResponse struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type JobScheduleResponse struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	TaskType        string     `json:"task_type"`
	PayloadJSON     *string    `json:"payload_json,omitempty"`
	IntervalMinutes int64      `json:"interval_minutes"`
	Enabled         bool       `json:"enabled"`
	NextRunAt       time.Time  `json:"next_run_at"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastJobID       *string    `json:"last_job_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
