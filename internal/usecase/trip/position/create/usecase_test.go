package create_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	positioncreate "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/position/create"
)

func TestExecute(t *testing.T) {
	ctx := context.Background()
	input := domaintrip.Position{
		TripID:      uuid.New(),
		Coordinates: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		RecordedAt:  time.Now().UTC(),
	}
	saved := input
	saved.ID = 42

	tests := []struct {
		name    string
		setup   func(*MocktripRepository)
		want    domaintrip.Position
		wantErr error
	}{
		{
			name: "active trip",
			setup: func(repo *MocktripRepository) {
				gomock.InOrder(
					repo.EXPECT().GetStatusForUpdate(ctx, input.TripID).Return(domaintrip.ActiveStatus, nil),
					repo.EXPECT().InsertPosition(ctx, input).Return(saved, nil),
					repo.EXPECT().UpdateLastPositionAt(ctx, input.TripID, input.RecordedAt).Return(nil),
				)
			},
			want: saved,
		},
		{
			name: "completed trip",
			setup: func(repo *MocktripRepository) {
				repo.EXPECT().GetStatusForUpdate(ctx, input.TripID).Return(domaintrip.CompletedStatus, nil)
			},
			wantErr: domaintrip.ErrTripCompleted,
		},
		{
			name: "trip not found",
			setup: func(repo *MocktripRepository) {
				repo.EXPECT().GetStatusForUpdate(ctx, input.TripID).Return(domaintrip.TripStatus(""), domaintrip.ErrTripNotFound)
			},
			wantErr: domaintrip.ErrTripNotFound,
		},
		{
			name: "insert fails",
			setup: func(repo *MocktripRepository) {
				gomock.InOrder(
					repo.EXPECT().GetStatusForUpdate(ctx, input.TripID).Return(domaintrip.ActiveStatus, nil),
					repo.EXPECT().InsertPosition(ctx, input).Return(domaintrip.Position{}, assert.AnError),
				)
			},
			wantErr: assert.AnError,
		},
		{
			name: "update fails",
			setup: func(repo *MocktripRepository) {
				gomock.InOrder(
					repo.EXPECT().GetStatusForUpdate(ctx, input.TripID).Return(domaintrip.ActiveStatus, nil),
					repo.EXPECT().InsertPosition(ctx, input).Return(saved, nil),
					repo.EXPECT().UpdateLastPositionAt(ctx, input.TripID, input.RecordedAt).Return(assert.AnError),
				)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			canceler := NewMockpositionCanceler(ctrl)
			if tt.wantErr == nil {
				canceler.EXPECT().Cancel(input.TripID)
			}

			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(
				func(txCtx context.Context, fn func(context.Context) error) error {
					return fn(txCtx)
				},
			)
			tt.setup(repo)

			got, err := positioncreate.New(repo, tx, canceler, func() time.Time { return input.RecordedAt }).Execute(ctx, input)

			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExecuteInvalidPositionSkipsTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	now := time.Now().UTC()
	position := domaintrip.Position{
		TripID:      uuid.New(),
		Coordinates: domaintrip.Point{Latitude: 91, Longitude: 37.61},
		RecordedAt:  now,
	}

	canceler := NewMockpositionCanceler(ctrl)

	_, err := positioncreate.New(repo, tx, canceler, func() time.Time { return now }).
		Execute(context.Background(), position)

	var validationErr *domaintrip.ValidationError
	if assert.ErrorAs(t, err, &validationErr) {
		assert.Equal(t, "coordinates.latitude", validationErr.Field)
	}
}
