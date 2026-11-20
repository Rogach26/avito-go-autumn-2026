package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}

type (
	staleTripRepository interface {
		ListStale(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error)
	}

	pushGateway interface {
		RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error
	}

	positionMetrics interface {
		StaleCheck(ctx context.Context, result string)
		QueueSizeChanged(delta int64)
		WorkerActiveChanged(delta int64)
		RequestCancelled(ctx context.Context)
		RequestFinished(ctx context.Context, result string, elapsed time.Duration)
	}

	ticker interface {
		C() <-chan time.Time
		Stop()
	}

	tickerFactory interface {
		NewTicker(interval time.Duration) ticker
	}
)
