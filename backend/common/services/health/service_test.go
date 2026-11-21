package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakePinger struct {
	err   error
	block bool
}

func (p fakePinger) Ping(ctx context.Context) error {
	if p.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return p.err
}

var unreachable = errors.New("connection refused")

func TestCheck_BothStoresAnswer(t *testing.T) {
	svc := New(fakePinger{}, fakePinger{}, "abc1234")

	report := svc.Check(context.Background())

	want := Report{Status: StatusOK, Mongo: StatusOK, Redis: StatusOK, Version: "abc1234"}
	if report != want {
		t.Errorf("report = %+v, want %+v", report, want)
	}
}

func TestCheck_WithoutRedisIsDegraded(t *testing.T) {
	svc := New(fakePinger{}, fakePinger{err: unreachable}, "")

	report := svc.Check(context.Background())

	if report.Status != StatusDegraded || report.Mongo != StatusOK || report.Redis != StatusDown {
		t.Errorf("report = %+v", report)
	}
}

func TestCheck_WithoutMongoIsDown(t *testing.T) {
	for name, redis := range map[string]fakePinger{"redis up": {}, "redis down": {err: unreachable}} {
		t.Run(name, func(t *testing.T) {
			svc := New(fakePinger{err: unreachable}, redis, "")

			report := svc.Check(context.Background())

			if report.Status != StatusDown || report.Mongo != StatusDown {
				t.Errorf("report = %+v", report)
			}
		})
	}
}

func TestCheck_AStoreThatHangsCountsAsDown(t *testing.T) {
	svc := New(fakePinger{block: true}, fakePinger{block: true}, "")
	svc.timeout = 20 * time.Millisecond

	start := time.Now()
	report := svc.Check(context.Background())
	elapsed := time.Since(start)

	if report.Status != StatusDown || report.Mongo != StatusDown || report.Redis != StatusDown {
		t.Errorf("report = %+v", report)
	}
	// Both stores are asked at once, so a hang costs one timeout, not two.
	if elapsed > 15*svc.timeout/10 {
		t.Errorf("check took %v with a %v timeout per store", elapsed, svc.timeout)
	}
}
