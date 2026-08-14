package safehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	REQUEST_TIMEOUT  = 10 * time.Second
	MAX_PAYLOAD_SIZE = 2 << 20 // 2mb
	MAX_REDIRECTS    = 3
)

var IsSafeIP = func(ip net.IP) bool {
	return !(ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified())
}

func Client() *http.Client {
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
