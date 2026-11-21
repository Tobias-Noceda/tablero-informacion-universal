package health

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
)

// PING_TIMEOUT is how long each store has to answer. An uptime monitor
// waits for the whole check, so a store that hangs must not hold it longer.
const PING_TIMEOUT = time.Second

type Status string

const (
	StatusOK       Status = "ok"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
)

// Report is what the health route answers. Version is the deployed commit,
// empty where the platform does not say.
type Report struct {
	Status  Status `json:"status"`
	Mongo   Status `json:"mongo"`
	Redis   Status `json:"redis"`
	Version string `json:"version,omitempty"`
}

type Service struct {
	mongo   infrastructure.Pinger
	redis   infrastructure.Pinger
	version string
	timeout time.Duration
}

func New(mongo, redis infrastructure.Pinger, version string) *Service {
	return &Service{mongo, redis, version, PING_TIMEOUT}
}

// Check asks both stores at once. Mongo holds every record, so without it
// the service is down. Without Redis it is degraded: whoever is signed in
// keeps working until their access token expires and cards run uncached,
// but nobody can sign in or renew a session.
func (srv *Service) Check(ctx context.Context) Report {
	report := Report{Version: srv.version}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		report.Mongo = srv.ping(ctx, "mongo", srv.mongo)
	}()
	go func() {
		defer wg.Done()
		report.Redis = srv.ping(ctx, "redis", srv.redis)
	}()
	wg.Wait()

	switch {
	case report.Mongo != StatusOK:
		report.Status = StatusDown
	case report.Redis != StatusOK:
		report.Status = StatusDegraded
	default:
		report.Status = StatusOK
	}
	return report
}

func (srv *Service) ping(ctx context.Context, name string, store infrastructure.Pinger) Status {
	ctx, cancel := context.WithTimeout(ctx, srv.timeout)
	defer cancel()

	if err := store.Ping(ctx); err != nil {
		log.Printf("health: %s: %v", name, err)
		return StatusDown
	}
	return StatusOK
}
