package httpapi

import "context"

type databasePinger interface {
	Ping(ctx context.Context) error
}
