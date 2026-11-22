package safehttp

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Every address a card could use to reach the platform's own network, or
// one that routes there, is refused; public addresses are not.
func TestIsSafeIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", // loopback
		"10.1.2.3", "172.16.0.1", "192.168.1.1", "fc00::1", // private
		"169.254.169.254", "fe80::1", // link-local, cloud metadata
		"0.0.0.0", "::", "0.1.2.3", // "this network": 0.x reaches the host itself on Linux
		"100.64.0.1", "100.127.255.254", // carrier-grade NAT, used by some cloud internals
		"192.0.0.8",                    // IETF protocol assignments
		"198.18.0.1", "198.19.255.254", // benchmarking
		"240.0.0.1", "255.255.255.255", // reserved and broadcast
		"224.0.0.1", "ff02::1", // multicast
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", // IPv4-mapped
		"64:ff9b::a00:1", "64:ff9b:1::a00:1", // NAT64 of 10.0.0.1
		"2002:a00:1::1",   // 6to4 of 10.0.0.1
		"2001:0:a00:1::1", // Teredo
	}
	for _, raw := range blocked {
		if IsSafeIP(net.ParseIP(raw)) {
			t.Errorf("%s is allowed, want it refused", raw)
		}
	}

	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "100.63.255.255", "100.128.0.1", "2001:4860:4860::8888", "2606:4700:4700::1111"}
	for _, raw := range allowed {
		if !IsSafeIP(net.ParseIP(raw)) {
			t.Errorf("%s is refused, want it allowed", raw)
		}
	}
}

// counting serves on loopback and counts what reached it.
func counting(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func ok(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

// The check runs where the connection is made, with the real IsSafeIP: no
// request ever reaches a loopback server, by address or by name.
func TestClient_NeverReachesALoopbackServer(t *testing.T) {
	srv, hits := counting(t, ok)
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))

	for _, target := range []string{srv.URL, "http://localhost:" + port} {
		res, err := newClient().Get(target)
		if err == nil {
			res.Body.Close()
			t.Errorf("%s was reachable", target)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("%d requests reached the loopback server", n)
	}
}

// allowOnlyLoopback lets the tests serve redirects locally, as the
// production guard would let a public host through.
func allowOnlyLoopback(t *testing.T) {
	t.Helper()
	original := IsSafeIP
	IsSafeIP = func(ip net.IP) bool { return ip.IsLoopback() }
	t.Cleanup(func() { IsSafeIP = original })
}

// A redirect may not send the request to another host, for instance from a
// public address to an internal one.
func TestClient_RedirectsStayOnTheSameHost(t *testing.T) {
	allowOnlyLoopback(t)
	inside, hits := counting(t, ok)
	outside, _ := counting(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, inside.URL+"/admin", http.StatusFound)
	})

	res, err := newClient().Get(outside.URL)
	if err == nil {
		res.Body.Close()
		t.Fatal("the cross-host redirect was followed")
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("the redirect target was reached %d times", n)
	}
}

// Two redirects on the same host are followed; a third is refused.
func TestClient_FollowsAtMostTwoRedirects(t *testing.T) {
	allowOnlyLoopback(t)
	srv, _ := counting(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1":
			http.Redirect(w, r, "/2", http.StatusFound)
		case "/2":
			http.Redirect(w, r, "/3", http.StatusFound)
		case "/3":
			http.Redirect(w, r, "/4", http.StatusFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	})

	res, err := newClient().Get(srv.URL + "/2")
	if err != nil {
		t.Fatalf("two redirects: %v", err)
	}
	res.Body.Close()

	if res, err := newClient().Get(srv.URL + "/1"); err == nil {
		res.Body.Close()
		t.Error("a third redirect was followed")
	}
}
