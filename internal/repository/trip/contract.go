package trip

import (
	"context"

	"github.com/exf1kzz/go-labs-avito/internal/infra/postgres/transaction"
)

type transactionManager interface {
	Executor(ctx context.Context) transaction.DBTX
}
