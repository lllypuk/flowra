package appcore

import (
	"context"
	"fmt"
)

// BaseUseCase contains common functionality for all use cases
type BaseUseCase struct{}

// WrapError wraps an error with context
func (b *BaseUseCase) WrapError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// ValidateContext checks that the context has not been canceled
func (b *BaseUseCase) ValidateContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
