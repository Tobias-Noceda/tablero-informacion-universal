package executer

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

func isSafeIP(ip net.IP) bool {
	return !(ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified())
}

func httpClient() *http.Client {
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
					if !isSafeIP(ip) {
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
			if len(via) >= 3 {
				return fmt.Errorf("Too many redirects")
			}

			if req.URL.Host != via[0].URL.Host {
				return fmt.Errorf("Cross-Host redirect not allowed")
			}

			return nil
		},
	}
}
