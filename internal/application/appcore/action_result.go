package appcore

import "github.com/lllypuk/flowra/internal/domain/uuid"

// ActionResult contains the result of an action
type ActionResult struct {
	MessageID uuid.UUID `json:"message_id"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}
