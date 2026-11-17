package get_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	getuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/get"
)

func TestExecute(t *testing.T) {
	tripID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	expectedTrip := domaintrip.Trip{
		ID:     tripID,
		Status: domaintrip.ActiveStatus,
	}

	tests := []struct {
		name    string
		setup   func(*gomock.Controller) *getuc.Usecase
		in      uuid.UUID
		want    domaintrip.Trip
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "trip found",
			setup: func(ctrl *gomock.Controller) *getuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				repo.EXPECT().GetByID(gomock.Any(), tripID).Return(expectedTrip, nil)
				return getuc.New(repo)
			},
			in:      tripID,
			want:    expectedTrip,
			wantErr: assert.NoError,
		},
		{
			name: "trip not found",
			setup: func(ctrl *gomock.Controller) *getuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				repo.EXPECT().GetByID(gomock.Any(), tripID).Return(domaintrip.Trip{}, domaintrip.ErrTripNotFound)
				return getuc.New(repo)
			},
			in:   tripID,
			want: domaintrip.Trip{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, domaintrip.ErrTripNotFound, msgAndArgs...)
			},
		},
		{
			name: "repository error",
			setup: func(ctrl *gomock.Controller) *getuc.Usecase {
				repo := NewMocktripRepository(ctrl)
				repo.EXPECT().GetByID(gomock.Any(), tripID).Return(domaintrip.Trip{}, assert.AnError)
				return getuc.New(repo)
			},
			in:   tripID,
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

			got, err := uc.Execute(context.Background(), tt.in)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
