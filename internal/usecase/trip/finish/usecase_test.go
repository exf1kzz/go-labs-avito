package finish_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	finishuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/finish"
)

func TestExecute(t *testing.T) {
	ctx := context.Background()
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	activeTrip := domaintrip.Trip{ID: id, Status: domaintrip.ActiveStatus}

	tests := []struct {
		name    string
		setup   func(*gomock.Controller) *finishuc.Usecase
		in      uuid.UUID
		want    domaintrip.Trip
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "trip completed",
			setup: func(ctrl *gomock.Controller) *finishuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				tx := NewMocktxManager(ctrl)
				expectTransaction(tx, ctx)

				var finishedAt time.Time
				gomock.InOrder(
					repo.EXPECT().GetByID(ctx, id).Return(activeTrip, nil),
					repo.EXPECT().Finish(
						ctx,
						id,
						gomock.Cond(func(at time.Time) bool {
							return !at.IsZero() && at.Location() == time.UTC
						}),
					).DoAndReturn(
						func(_ context.Context, _ uuid.UUID, at time.Time) (domaintrip.Trip, error) {
							finishedAt = at
							updated := activeTrip
							updated.Status = domaintrip.CompletedStatus
							updated.FinishedAt = &at
							return updated, nil
						},
					),
					repo.EXPECT().AddStatusHistory(
						ctx,
						id,
						gomock.Cond(func(from *domaintrip.TripStatus) bool {
							return from != nil && *from == domaintrip.ActiveStatus
						}),
						domaintrip.CompletedStatus,
						nil,
						gomock.Cond(func(at time.Time) bool {
							return at.Equal(finishedAt)
						}),
					).Return(nil),
				)

				return finishuc.New(repo, tx)
			},
			in:      id,
			want:    domaintrip.Trip{ID: id, Status: domaintrip.CompletedStatus},
			wantErr: assert.NoError,
		},
		{
			name: "already completed",
			setup: func(ctrl *gomock.Controller) *finishuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				tx := NewMocktxManager(ctrl)
				expectTransaction(tx, ctx)

				repo.EXPECT().GetByID(ctx, id).Return(
					domaintrip.Trip{ID: id, Status: domaintrip.CompletedStatus}, nil,
				)
				return finishuc.New(repo, tx)
			},
			in:   id,
			want: domaintrip.Trip{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, domaintrip.ErrTripCompleted, msgAndArgs...)
			},
		},
		{
			name: "trip not found",
			setup: func(ctrl *gomock.Controller) *finishuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				tx := NewMocktxManager(ctrl)
				expectTransaction(tx, ctx)

				repo.EXPECT().GetByID(ctx, id).Return(domaintrip.Trip{}, domaintrip.ErrTripNotFound)
				return finishuc.New(repo, tx)
			},
			in:   id,
			want: domaintrip.Trip{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, domaintrip.ErrTripNotFound, msgAndArgs...)
			},
		},
		{
			name: "repository error",
			setup: func(ctrl *gomock.Controller) *finishuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				tx := NewMocktxManager(ctrl)
				expectTransaction(tx, ctx)

				gomock.InOrder(
					repo.EXPECT().GetByID(ctx, id).Return(activeTrip, nil),
					repo.EXPECT().Finish(
						ctx,
						id,
						gomock.Cond(func(at time.Time) bool {
							return !at.IsZero() && at.Location() == time.UTC
						}),
					).Return(domaintrip.Trip{}, assert.AnError),
				)
				return finishuc.New(repo, tx)
			},
			in:   id,
			want: domaintrip.Trip{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError, msgAndArgs...)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := tt.setup(ctrl)

			got, err := uc.Execute(ctx, tt.in)

			tt.wantErr(t, err)
			if err == nil && assert.NotNil(t, got.FinishedAt) {
				assert.False(t, got.FinishedAt.IsZero())
				assert.Equal(t, time.UTC, got.FinishedAt.Location())
				got.FinishedAt = nil
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func expectTransaction(tx *MocktxManager, ctx context.Context) {
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(
		func(txCtx context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		},
	)
}
