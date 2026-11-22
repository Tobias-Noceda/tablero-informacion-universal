package safehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	REQUEST_TIMEOUT  = 10 * time.Second
	MAX_PAYLOAD_SIZE = 2 << 20 // 2mb
	MAX_REDIRECTS    = 3
)

// blockedNetworks are the special ranges the net package does not name that
// still reach the host, its network or an internal one, or embed an IPv4
// address that might.
var blockedNetworks = parseNetworks(
	"0.0.0.0/8",      // "this network": 0.x reaches the host itself on Linux
	"100.64.0.0/10",  // carrier-grade NAT, used by some cloud internals
	"192.0.0.0/24",   // IETF protocol assignments
	"198.18.0.0/15",  // benchmarking
	"240.0.0.0/4",    // reserved, and the broadcast address
	"64:ff9b::/96",   // NAT64, well-known prefix
	"64:ff9b:1::/48", // NAT64, local-use prefix
	"2002::/16",      // 6to4, embeds an IPv4 address
	"2001::/32",      // Teredo, embeds an IPv4 address
)

func parseNetworks(cidrs ...string) []*net.IPNet {
	networks := make([]*net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		networks[i] = network
	}
	return networks
}

// IsSafeIP is whether a card may connect to ip. A variable so tests can serve
// on loopback; production never changes it.
var IsSafeIP = func(ip net.IP) bool {
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return false
	}
	for _, network := range blockedNetworks {
		if network.Contains(ip) {
			return false
		}
	}
	return true
}

var (
	clientOnce   sync.Once
	sharedClient *http.Client
)

func Client() *http.Client {
	clientOnce.Do(func() {
		sharedClient = newClient()
	})

	return sharedClient
}

func newClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}

				ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if err != nil {
					return nil, err
				}

				dialer := &net.Dialer{
					Timeout: REQUEST_TIMEOUT,
				}

				for _, ip := range ips {
					if !IsSafeIP(ip) {
						continue
					}

					conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
					if err == nil {
						return conn, nil
					}
				}

				return nil, fmt.Errorf("Failed to connect")
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MAX_REDIRECTS {
				return fmt.Errorf("Too many redirects")
			}

			if req.URL.Host != via[0].URL.Host {
				return fmt.Errorf("Cross-Host redirect not allowed")
			}

			return nil
		},
	}
}
