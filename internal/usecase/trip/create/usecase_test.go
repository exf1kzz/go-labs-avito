package create_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/model"
	createuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/create"
)

func TestExecute(t *testing.T) {
	ctx := context.Background()
	input := model.CreateTripInput{
		UserID:     uuid.MustParse("00000000-0000-4000-8000-000000000001"),
		DriverID:   uuid.MustParse("00000000-0000-4000-8000-000000000002"),
		StartPoint: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   domaintrip.Point{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
	}
	expectedTrip := domaintrip.Trip{
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     domaintrip.ActiveStatus,
	}

	tests := []struct {
		name    string
		setup   func(*gomock.Controller) *createuc.Usecase
		in      model.CreateTripInput
		want    domaintrip.Trip
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "trip created",
			setup: func(ctrl *gomock.Controller) *createuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				tx := NewMocktxManager(ctrl)
				expectTransaction(tx, ctx)

				var saved domaintrip.Trip
				gomock.InOrder(
					repo.EXPECT().
						Create(ctx, createTripMatcher(input)).
						DoAndReturn(func(_ context.Context, trip domaintrip.Trip) error {
							saved = trip
							return nil
						}),
					repo.EXPECT().AddStatusHistory(
						ctx,
						gomock.Cond(func(id uuid.UUID) bool {
							return id == saved.ID
						}),
						nil,
						domaintrip.ActiveStatus,
						nil,
						gomock.Cond(func(at time.Time) bool {
							return at.Equal(saved.StartedAt)
						}),
					).Return(nil),
				)

				return createuc.New(repo, tx)
			},
			in:      input,
			want:    expectedTrip,
			wantErr: assert.NoError,
		},
		{
			name: "driver busy",
			setup: func(ctrl *gomock.Controller) *createuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				tx := NewMocktxManager(ctrl)
				expectTransaction(tx, ctx)

				repo.EXPECT().Create(ctx, createTripMatcher(input)).Return(domaintrip.ErrDriverBusy)

				return createuc.New(repo, tx)
			},
			in:   input,
			want: domaintrip.Trip{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, domaintrip.ErrDriverBusy, msgAndArgs...)
			},
		},
		{
			name: "repository error",
			setup: func(ctrl *gomock.Controller) *createuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				tx := NewMocktxManager(ctrl)
				expectTransaction(tx, ctx)

				repo.EXPECT().Create(ctx, createTripMatcher(input)).Return(assert.AnError)

				return createuc.New(repo, tx)
			},
			in:   input,
			want: domaintrip.Trip{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError, msgAndArgs...)
			},
		},
		{
			name: "invalid input",
			setup: func(ctrl *gomock.Controller) *createuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				tx := NewMocktxManager(ctrl)
				return createuc.New(repo, tx)
			},
			in:   model.CreateTripInput{},
			want: domaintrip.Trip{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				var validationErr *domaintrip.ValidationError
				if !assert.ErrorAs(t, err, &validationErr, msgAndArgs...) {
					return false
				}
				return assert.Equal(t, "user_id", validationErr.Field)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := tt.setup(ctrl)

			got, err := uc.Execute(ctx, tt.in)

			tt.wantErr(t, err)
			if err == nil {
				assert.NotEqual(t, uuid.Nil, got.ID)
				assert.False(t, got.StartedAt.IsZero())
				assert.Equal(t, time.UTC, got.StartedAt.Location())

				got.ID = uuid.Nil
				got.StartedAt = time.Time{}
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

func createTripMatcher(input model.CreateTripInput) gomock.Matcher {
	expected := domaintrip.Trip{
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     domaintrip.ActiveStatus,
	}

	return gomock.Cond(func(got domaintrip.Trip) bool {
		if got.ID == uuid.Nil || got.StartedAt.IsZero() ||
			got.StartedAt.Location() != time.UTC {
			return false
		}

		got.ID = uuid.Nil
		got.StartedAt = time.Time{}
		return got == expected
	})
}
