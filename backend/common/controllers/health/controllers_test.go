package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	srv "github.com/Secreto31126/tesis/common/services/health"
	"github.com/gin-gonic/gin"
)

type pinger struct{ err error }

func (p pinger) Ping(context.Context) error { return p.err }

var unreachable = errors.New("connection refused")

func get(mongo, redis pinger) *httptest.ResponseRecorder {
	return send(http.MethodGet, mongo, redis)
}

func send(method string, mongo, redis pinger) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(srv.New(mongo, redis, "abc1234")).RegisterRoutes(r.Group("/api/v1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, "/api/v1/health", nil))
	return w
}

// Uptime monitors such as UptimeRobot check with HEAD by default: it must
// answer the same status as GET, without a body.
func TestHealth_AnswersHEAD(t *testing.T) {
	for name, c := range map[string]struct {
		mongo pinger
		want  int
	}{
		"up":   {pinger{}, http.StatusOK},
		"down": {pinger{err: unreachable}, http.StatusServiceUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			w := send(http.MethodHead, c.mongo, pinger{})

			if w.Code != c.want {
				t.Errorf("HEAD status = %d, want %d", w.Code, c.want)
			}
			if w.Body.Len() != 0 {
				t.Errorf("HEAD answered a body: %q", w.Body.String())
			}
			if cache := w.Header().Get("Cache-Control"); cache != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", cache)
			}
		})
	}
}

func decode(t *testing.T, w *httptest.ResponseRecorder) srv.Report {
	t.Helper()
	var report srv.Report
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("body %q: %v", w.Body.String(), err)
	}
	return report
}

func TestHealth_OK(t *testing.T) {
	w := get(pinger{}, pinger{})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if report := decode(t, w); report != (srv.Report{Status: srv.StatusOK, Mongo: srv.StatusOK, Redis: srv.StatusOK, Version: "abc1234"}) {
		t.Errorf("report = %+v", report)
	}
	if cache := w.Header().Get("Cache-Control"); cache != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cache)
	}
}

// A degraded service still answers 200: it is up, and an uptime monitor
// must not count it as an outage. The body says what is missing.
func TestHealth_DegradedIsStillUp(t *testing.T) {
	w := get(pinger{}, pinger{err: unreachable})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if report := decode(t, w); report.Status != srv.StatusDegraded || report.Redis != srv.StatusDown {
		t.Errorf("report = %+v", report)
	}
}

func TestHealth_DownAnswers503(t *testing.T) {
	w := get(pinger{err: unreachable}, pinger{})

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if report := decode(t, w); report.Status != srv.StatusDown || report.Mongo != srv.StatusDown {
		t.Errorf("report = %+v", report)
	}
}

func TestHealth_NeverSaysWhyAStoreFailed(t *testing.T) {
	w := get(pinger{err: errors.New("auth failed for mongodb://user:secret@db")}, pinger{})

	if body := w.Body.String(); strings.Contains(body, "secret") || strings.Contains(body, "mongodb://") {
		t.Errorf("the error leaked into the response: %s", body)
	}
}
