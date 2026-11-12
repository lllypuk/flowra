package appcore

import "github.com/lllypuk/flowra/internal/domain/uuid"

// ActionResult is the outcome of an applied chat action.
type ActionResult struct {
	ChatID   uuid.UUID `json:"chat_id"`
	Version  int       `json:"version"`
	Replayed bool      `json:"replayed"`
	Success  bool      `json:"success"`
	Error    string    `json:"error,omitempty"`
}
