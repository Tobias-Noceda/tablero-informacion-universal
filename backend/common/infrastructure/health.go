package infrastructure

import "context"

// Pinger is a store that can say whether it answers. The health check asks
// Mongo and Redis through it.
type Pinger interface {
	Ping(ctx context.Context) error
}
