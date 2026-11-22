//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

// A card of one's own cannot reach the platform's network. During
// these tests Mongo and Redis listen on the host's loopback, which makes
// them a real target, not a stand-in.
func TestEndToEnd_ACardCannotReachInternalAddresses(t *testing.T) {
	s := newStack(t)
	owner := s.signUp("owner")
	board := owner.newBoard("ssrf")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)

	for _, target := range []string{
		"http://127.0.0.1:27017/",                  // Mongo
		"http://localhost:6379/",                   // Redis, by name
		"http://[::1]:27017/",                      // loopback over IPv6
		"http://0.0.0.0:27017/",                    // "this host"
		"http://169.254.169.254/latest/meta-data/", // cloud metadata
		"http://100.64.0.1/",                       // carrier-grade NAT
		"http://10.0.0.1/",                         // private network
		"http://2130706433/",                       // 127.0.0.1 written as a number
	} {
		t.Run(target, func(t *testing.T) {
			var postit models.PostIts
			s.mustJSON(owner.do(http.MethodPost, "/api/v1/post-its", map[string]any{
				"board":    board.Id,
				"title":    map[string]any{"text": "probe", "vars": false},
				"resource": target,
				"request":  map[string]any{"method": http.MethodGet},
				"response": "json",
				"query":    map[string]string{"value": "."},
				"rate":     60,
			}), http.StatusCreated, &postit)

			w := owner.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil)

			var body map[string]any
			_ = json.Unmarshal(w.Body.Bytes(), &body)
			if w.Code != http.StatusBadGateway || body["code"] != "source_unreachable" {
				t.Errorf("answered %d %s, want 502 source_unreachable", w.Code, w.Body.String())
			}
			// Nothing about the internal service leaks into the answer.
			if strings.Contains(strings.ToLower(w.Body.String()), "mongo") || strings.Contains(w.Body.String(), "redis_version") {
				t.Errorf("the answer carries what the target said: %s", w.Body.String())
			}
		})
	}
}
