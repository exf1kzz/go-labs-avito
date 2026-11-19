package positions

import (
	"context"
	"time"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=positions_test
type staleRepository interface {
	ListStaleActive(ctx context.Context, duration time.Duration, batchSize uint64) ([]uuid.UUID, error)
}

type jobQueue interface {
	TrySubmit(uuid.UUID) bool
}
