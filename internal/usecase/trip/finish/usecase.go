package finish

import (
	"context"
	"fmt"
	"time"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/google/uuid"
)

type Usecase struct {
	repository tripRepository
	txManager  txManager
}

func New(repository tripRepository, txManager txManager) *Usecase {
	return &Usecase{
		repository: repository,
		txManager:  txManager,
	}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	finishedAt := time.Now().UTC()
	var updated domaintrip.Trip

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		current, err := u.repository.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		fromStatus := current.Status
		if err := current.Finish(finishedAt); err != nil {
			return err
		}

		updated, err = u.repository.Finish(txCtx, id, finishedAt)
		if err != nil {
			return err
		}

		return u.repository.AddStatusHistory(
			txCtx, id, &fromStatus, current.Status, nil, finishedAt,
		)
	})
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	return updated, nil
}
