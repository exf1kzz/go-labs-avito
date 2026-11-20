package push_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/exf1kzz/go-labs-avito/internal/gateway/push"
)

func TestRequestPosition(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := NewMockhttpDoer(ctrl)
	driverID := uuid.New()
	tripID := uuid.New()

	client.EXPECT().Do(gomock.Any()).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "http://push:8090/api/v1/push", req.URL.String())
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

			var body struct {
				RequestID   string            `json:"request_id"`
				RecipientID uuid.UUID         `json:"recipient_id"`
				Kind        string            `json:"kind"`
				Data        map[string]string `json:"data"`
			}
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			_, err := uuid.Parse(body.RequestID)
			require.NoError(t, err)
			assert.Equal(t, driverID, body.RecipientID)
			assert.Equal(t, "REQUEST_POSITION", body.Kind)
			assert.Equal(t, tripID.String(), body.Data["trip_id"])

			return &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}, nil
		},
	)

	gateway, err := push.New("http://push:8090", client)
	require.NoError(t, err)
	require.NoError(t, gateway.RequestPosition(context.Background(), driverID, tripID))
}

func TestRequestPositionFailures(t *testing.T) {
	t.Run("HTTP error", func(t *testing.T) {
		client := NewMockhttpDoer(gomock.NewController(t))
		client.EXPECT().Do(gomock.Any()).Return(&http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("unavailable")),
		}, nil)

		gateway, err := push.New("http://push:8090", client)
		require.NoError(t, err)
		err = gateway.RequestPosition(context.Background(), uuid.New(), uuid.New())
		assert.ErrorContains(t, err, "HTTP 503")
	})

	t.Run("transport error", func(t *testing.T) {
		client := NewMockhttpDoer(gomock.NewController(t))
		client.EXPECT().Do(gomock.Any()).Return(nil, assert.AnError)

		gateway, err := push.New("http://push:8090", client)
		require.NoError(t, err)
		err = gateway.RequestPosition(context.Background(), uuid.New(), uuid.New())
		assert.True(t, errors.Is(err, assert.AnError))
	})
}
