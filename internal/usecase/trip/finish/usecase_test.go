package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	finishuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/finish"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestExecuteSuccess(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	current := domaintrip.Trip{ID: id, Status: domaintrip.ActiveStatus}

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)

	var finishedAt time.Time
	gomock.InOrder(
		repo.EXPECT().GetByID(ctx, id).Return(current, nil),
		repo.EXPECT().Finish(ctx, id, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ uuid.UUID, at time.Time) (domaintrip.Trip, error) {
				finishedAt = at
				updated := current
				updated.Status = domaintrip.CompletedStatus
				updated.FinishedAt = &at
				return updated, nil
			},
		),
		repo.EXPECT().AddStatusHistory(
			ctx, id, gomock.Any(), domaintrip.CompletedStatus, nil, gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ uuid.UUID, from *domaintrip.TripStatus,
				_ domaintrip.TripStatus, _ *string, at time.Time) error {
				if from == nil || *from != domaintrip.ActiveStatus || !at.Equal(finishedAt) {
					t.Errorf("history transition or time is incorrect")
				}
				return nil
			},
		),
	)

	got, err := finishuc.New(repo, tx).Execute(ctx, id)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.ID != id || got.Status != domaintrip.CompletedStatus ||
		got.FinishedAt == nil || !got.FinishedAt.Equal(finishedAt) {
		t.Errorf("completed trip = %+v", got)
	}
}

func TestExecuteAlreadyCompleted(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)
	repo.EXPECT().GetByID(ctx, id).Return(
		domaintrip.Trip{ID: id, Status: domaintrip.CompletedStatus}, nil,
	)

	got, err := finishuc.New(repo, tx).Execute(ctx, id)
	if !errors.Is(err, domaintrip.ErrTripCompleted) {
		t.Fatalf("Execute() error = %v, want ErrTripCompleted", err)
	}
	if got != (domaintrip.Trip{}) {
		t.Errorf("trip on error = %+v, want zero value", got)
	}
}
