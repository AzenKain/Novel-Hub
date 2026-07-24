package response

import "time"

type BackupResponse struct {
	Name         string    `json:"name"`
	SizeBytes    int64     `json:"sizeBytes"`
	CreatedAt    time.Time `json:"createdAt"`
	IncludeBooks bool      `json:"includeBooks"`
}

type RestoreResponse struct {
	RestartRequired bool `json:"restartRequired"`
	AutoRestart     bool `json:"autoRestart"`
}

type LogFileResponse struct {
	Name      string    `json:"name"`
	SizeBytes int64     `json:"sizeBytes"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type LogTailResponse struct {
	File  string   `json:"file"`
	Lines []string `json:"lines"`
}
