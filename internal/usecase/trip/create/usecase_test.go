package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/model"
	createuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/create"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestExecuteSuccess(t *testing.T) {
	ctx := context.Background()
	input := model.CreateTripInput{
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   domaintrip.Point{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
	}

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)

	var saved domaintrip.Trip
	gomock.InOrder(
		repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, trip domaintrip.Trip) error {
				saved = trip
				if trip.ID == uuid.Nil || trip.UserID != input.UserID ||
					trip.DriverID != input.DriverID || trip.Price != input.Price ||
					trip.Status != domaintrip.ActiveStatus {
					t.Errorf("saved trip has wrong fields: %+v", trip)
				}
				return nil
			},
		),
		repo.EXPECT().AddStatusHistory(
			ctx, gomock.Any(), nil, domaintrip.ActiveStatus, nil, gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, id uuid.UUID, _ *domaintrip.TripStatus,
				_ domaintrip.TripStatus, _ *string, at time.Time) error {
				if id != saved.ID || !at.Equal(saved.StartedAt) {
					t.Errorf("history does not match saved trip")
				}
				return nil
			},
		),
	)

	got, err := createuc.New(repo, tx).Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.ID != saved.ID {
		t.Errorf("returned ID = %s, saved ID = %s", got.ID, saved.ID)
	}
}

func TestExecuteDriverBusy(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)
	repo.EXPECT().Create(ctx, gomock.Any()).Return(domaintrip.ErrDriverBusy)

	got, err := createuc.New(repo, tx).Execute(ctx, model.CreateTripInput{
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		Price:    100,
	})
	if !errors.Is(err, domaintrip.ErrDriverBusy) {
		t.Fatalf("Execute() error = %v, want ErrDriverBusy", err)
	}
	if got != (domaintrip.Trip{}) {
		t.Errorf("trip on error = %+v, want zero value", got)
	}
}

func TestExecuteInvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	got, err := createuc.New(repo, tx).Execute(
		context.Background(), model.CreateTripInput{},
	)

	var validationErr *domaintrip.ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != "user_id" {
		t.Fatalf("error = %v, want validation error for user_id", err)
	}
	if got != (domaintrip.Trip{}) {
		t.Errorf("trip on error = %+v, want zero value", got)
	}
}
