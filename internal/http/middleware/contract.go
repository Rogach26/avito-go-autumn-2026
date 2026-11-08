package middleware

import (
	"context"
	"time"
)

type MetricsRecorder interface {
	RequestStarted(ctx context.Context, method, route string)
	RequestFinished(ctx context.Context, method, route string, status int, duration time.Duration)
}
