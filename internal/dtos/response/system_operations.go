package response

import "time"

type BackupResponse struct {
	Name         string    `json:"name"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
	IncludeBooks bool      `json:"include_books"`
}

type RestoreResponse struct {
	RestartRequired bool `json:"restart_required"`
	AutoRestart     bool `json:"auto_restart"`
}

type LogFileResponse struct {
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LogTailResponse struct {
	File  string   `json:"file"`
	Lines []string `json:"lines"`
}
