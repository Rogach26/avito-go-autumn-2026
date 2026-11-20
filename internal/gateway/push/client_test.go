package push

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestClientRequestPosition(t *testing.T) {
	t.Parallel()

	driverID := uuid.New()
	tripID := uuid.New()
	client := newTestClient(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/v1/push" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", request.Header.Get("Content-Type"))
		}

		var body struct {
			RequestID   string            `json:"request_id"`
			RecipientID uuid.UUID         `json:"recipient_id"`
			Kind        string            `json:"kind"`
			Data        map[string]string `json:"data"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.RequestID == "" || body.RecipientID != driverID || body.Kind != "REQUEST_POSITION" || body.Data["trip_id"] != tripID.String() {
			t.Errorf("unexpected request body: %+v", body)
		}
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, nil
	}))

	if err := client.RequestPosition(t.Context(), driverID, tripID); err != nil {
		t.Fatalf("request position: %v", err)
	}
}

func TestClientRequestPositionHTTPError(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, nil
	}))

	err := client.RequestPosition(t.Context(), uuid.New(), uuid.New())
	if err == nil || !strings.Contains(err.Error(), "server error: HTTP status 503") {
		t.Fatalf("request position: got %v, want HTTP server error", err)
	}
}

func TestClientRequestPositionTransportError(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}))

	err := client.RequestPosition(t.Context(), uuid.New(), uuid.New())
	if err == nil || !strings.Contains(err.Error(), "send push request") {
		t.Fatalf("request position: got %v, want transport error", err)
	}
}

func newTestClient(t *testing.T, transport http.RoundTripper) *Client {
	t.Helper()
	client, err := New("http://push-service", time.Second, noop.NewTracerProvider())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	client.client.Transport = transport
	t.Cleanup(client.CloseIdleConnections)
	return client
}
