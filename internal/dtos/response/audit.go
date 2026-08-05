package response

import "time"

type AuditLogResponse struct {
	ID          string    `json:"id"`
	ActorID     *string   `json:"actor_id,omitempty"`
	ActorEmail  string    `json:"actor_email"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    *string   `json:"target_id,omitempty"`
	TargetLabel string    `json:"target_label"`
	IP          string    `json:"ip"`
	CreatedAt   time.Time `json:"created_at"`
}
