package list_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	listuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/position/list"
)

func TestExecute(t *testing.T) {
	ctx := context.Background()
	tripID := uuid.New()
	trip := domaintrip.Trip{ID: tripID}
	positions := []domaintrip.Position{{
		ID:         42,
		TripID:     tripID,
		RecordedAt: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
	}}

	tests := []struct {
		name    string
		setup   func(*MocktripRepository)
		want    []domaintrip.Position
		wantErr error
	}{
		{
			name: "route with positions",
			setup: func(repo *MocktripRepository) {
				gomock.InOrder(
					repo.EXPECT().GetByID(ctx, tripID).Return(trip, nil),
					repo.EXPECT().ListPositions(ctx, tripID).Return(positions, nil),
				)
			},
			want: positions,
		},
		{
			name: "empty route",
			setup: func(repo *MocktripRepository) {
				gomock.InOrder(
					repo.EXPECT().GetByID(ctx, tripID).Return(trip, nil),
					repo.EXPECT().ListPositions(ctx, tripID).
						Return([]domaintrip.Position{}, nil),
				)
			},
			want: []domaintrip.Position{},
		},
		{
			name: "trip not found",
			setup: func(repo *MocktripRepository) {
				repo.EXPECT().GetByID(ctx, tripID).
					Return(domaintrip.Trip{}, domaintrip.ErrTripNotFound)
			},
			wantErr: domaintrip.ErrTripNotFound,
		},
		{
			name: "list fails",
			setup: func(repo *MocktripRepository) {
				gomock.InOrder(
					repo.EXPECT().GetByID(ctx, tripID).Return(trip, nil),
					repo.EXPECT().ListPositions(ctx, tripID).
						Return(nil, assert.AnError),
				)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tt.setup(repo)

			got, err := listuc.New(repo).Execute(ctx, tripID)

			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
