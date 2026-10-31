package get_test

import (
	"context"
	"errors"
	"testing"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	getuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/get"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestExecute(t *testing.T) {
	id := uuid.New()
	wantTrip := domaintrip.Trip{ID: id, Status: domaintrip.ActiveStatus}
	dbErr := errors.New("database unavailable")

	tests := []struct {
		name     string
		repoTrip domaintrip.Trip
		repoErr  error
		wantErr  error
	}{
		{name: "trip found", repoTrip: wantTrip},
		{name: "trip not found", repoErr: domaintrip.ErrTripNotFound, wantErr: domaintrip.ErrTripNotFound},
		{name: "repository error", repoErr: dbErr, wantErr: dbErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			repo.EXPECT().GetByID(gomock.Any(), id).Return(tt.repoTrip, tt.repoErr)

			got, err := getuc.New(repo).Execute(context.Background(), id)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got.ID != wantTrip.ID {
				t.Errorf("trip ID = %s, want %s", got.ID, wantTrip.ID)
			}
			if tt.wantErr != nil && got != (domaintrip.Trip{}) {
				t.Errorf("trip on error = %+v, want zero value", got)
			}
		})
	}
}
