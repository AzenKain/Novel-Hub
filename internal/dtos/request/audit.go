package request

type ListAuditLogsDto struct {
	PaginationDto
	Action  string `json:"action,omitempty" query:"action" validate:"omitempty,max=64"`
	ActorID string `json:"actor_id,omitempty" query:"actor_id" validate:"omitempty,uuid"`
}
