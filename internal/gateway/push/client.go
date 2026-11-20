package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	url    string
	client *http.Client
}

type request struct {
	RequestID   string            `json:"request_id"`
	RecipientID uuid.UUID         `json:"recipient_id"`
	Kind        string            `json:"kind"`
	Data        map[string]string `json:"data"`
}

func New(baseURL string, timeout time.Duration, tracerProvider trace.TracerProvider) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse push service URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("parse push service URL: absolute URL is required")
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/api/v1/push"
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	transport := http.DefaultTransport.(*http.Transport).Clone()
	return &Client{
		url: parsedURL.String(),
		client: &http.Client{
			Timeout: timeout,
			Transport: otelhttp.NewTransport(
				transport,
				otelhttp.WithTracerProvider(tracerProvider),
				otelhttp.WithSpanNameFormatter(func(_ string, _ *http.Request) string {
					return "push.request_position"
				}),
			),
		},
	}, nil
}

func (c *Client) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	body, err := json.Marshal(request{
		RequestID:   uuid.NewString(),
		RecipientID: driverID,
		Kind:        "REQUEST_POSITION",
		Data:        map[string]string{"trip_id": tripID.String()},
	})
	if err != nil {
		return fmt.Errorf("encode push request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("send push request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("push service server error: HTTP status %d", response.StatusCode)
	}
	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("push service client error: HTTP status %d", response.StatusCode)
	}
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("push service returned unexpected HTTP status %d", response.StatusCode)
	}
	return nil
}

func (c *Client) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}
