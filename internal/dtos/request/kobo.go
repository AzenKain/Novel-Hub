package request

type KoboSyncStateDto struct {
	State map[string]any `json:"state" validate:"required"`
}
