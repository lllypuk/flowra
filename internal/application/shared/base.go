package shared

import (
	"context"
	"fmt"
)

// BaseUseCase содержит общую функциональность для всех use cases
type BaseUseCase struct{}

// WrapError оборачивает ошибку с контекстом
func (b *BaseUseCase) WrapError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// ValidateContext проверяет, что контекст не был отменен
func (b *BaseUseCase) ValidateContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
