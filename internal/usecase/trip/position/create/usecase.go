package create

import (
	"context"
	"fmt"
	"time"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

type Usecase struct {
	repository tripRepository
	txManager  txManager
	now        func() time.Time
}

func New(repository tripRepository, txManager txManager, now func() time.Time) *Usecase {
	return &Usecase{
		repository: repository,
		txManager:  txManager,
		now:        now,
	}
}

func (u *Usecase) Execute(
	ctx context.Context,
	position domaintrip.Position,
) (domaintrip.Position, error) {
	if err := position.Validate(u.now()); err != nil {
		return domaintrip.Position{}, fmt.Errorf("validate position: %w", err)
	}

	var saved domaintrip.Position

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		status, err := u.repository.GetStatusForUpdate(txCtx, position.TripID)
		if err != nil {
			return err
		}
		if status != domaintrip.ActiveStatus {
			return domaintrip.ErrTripCompleted
		}

		saved, err = u.repository.InsertPosition(txCtx, position)
		if err != nil {
			return err
		}

		return u.repository.UpdateLastPositionAt(
			txCtx, position.TripID, position.RecordedAt,
		)
	})
	if err != nil {
		return domaintrip.Position{}, fmt.Errorf("create trip position: %w", err)
	}

	return saved, nil
}
