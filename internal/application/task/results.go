package task

import (
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// TaskResult — result vypolneniya use case for Task
// imya yavno ukazyvaet on prinadlezhnost to task for izbezhaniya putanitsy s drugimi rezultatami
//
//nolint:revive // osoznannoe reshenie for yasnosti koda
type TaskResult struct {
	// TaskID identifier tasks
	TaskID uuid.UUID

	// Version tekuschaya versiya aggregate after vypolneniya operatsii
	Version int

	// Success flag uspeshnogo vypolneniya
	Success bool

	// Message dopolnitelnoe message (for errors or preduprezhdeniy)
	Message string
}

// NewSuccessResult creates result uspeshnogo vypolneniya
func NewSuccessResult(taskID uuid.UUID, version int) TaskResult {
	return TaskResult{
		TaskID:  taskID,
		Version: version,
		Success: true,
	}
}

// NewFailureResult creates result neudachnogo vypolneniya
func NewFailureResult(taskID uuid.UUID, message string) TaskResult {
	return TaskResult{
		TaskID:  taskID,
		Success: false,
		Message: message,
	}
}

// IsSuccess returns true if operatsiya vypolnena successfully
func (r TaskResult) IsSuccess() bool {
	return r.Success
}

// IsFailure returns true if operatsiya zavershilas s oshibkoy
func (r TaskResult) IsFailure() bool {
	return !r.Success
}
