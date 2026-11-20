package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

//go:generate go tool mockgen -source=gateway.go -destination=mocks_test.go -package=push_test
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Gateway struct {
	endpoint string
	client   httpDoer
}

func New(baseURL string, client httpDoer) (*Gateway, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse push URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("invalid push URL %q", baseURL)
	}

	endpoint, err := url.JoinPath(baseURL, "api/v1/push")
	if err != nil {
		return nil, fmt.Errorf("build push endpoint: %w", err)
	}

	return &Gateway{
		endpoint: endpoint,
		client:   client,
	}, nil
}

func (g *Gateway) RequestPosition(
	ctx context.Context,
	driverID uuid.UUID,
	tripID uuid.UUID,
) error {
	body, err := json.Marshal(struct {
		RequestID   string            `json:"request_id"`
		RecipientID uuid.UUID         `json:"recipient_id"`
		Kind        string            `json:"kind"`
		Data        map[string]string `json:"data"`
	}{
		RequestID:   uuid.NewString(),
		RecipientID: driverID,
		Kind:        "REQUEST_POSITION",
		Data:        map[string]string{"trip_id": tripID.String()},
	})
	if err != nil {
		return fmt.Errorf("encode position request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, g.endpoint, bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build position request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("send position request: %w", err)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("push service returned HTTP %d", resp.StatusCode)
	}
	return nil
}
