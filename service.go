package tsweb

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

type Service struct {
	*tsnet.Server
}

func Start(s *tsnet.Server) (*Service, error) {
	// ensure state dir exists
	if d := s.Dir; d != "" {
		if _, err := os.Stat(d); err != nil {
			if err := os.MkdirAll(d, 0700); err != nil {
				return nil, err
			}
		}
	}

	if err := s.Start(); err != nil {
		return nil, err
	}

	return &Service{
		Server: s,
	}, nil
}

func (s *Service) WaitUntilReady(ctx context.Context) error {
	c, err := s.LocalClient()
	if err != nil {
		return err
	}

	if _, err := waitUntilReady(ctx, c); err != nil {
		return err
	}

	return nil
}

func waitUntilReady(
	ctx context.Context,
	c *tailscale.LocalClient,
) (*ipnstate.Status, error) {
	for {
		status, err := c.Status(ctx)
		if err != nil {
			return nil, err
		}

		if status.BackendState == "Running" {
			return status, nil
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Service) ListenTLS(network string, addr string) (net.Listener, error) {
	c, err := s.LocalClient()
	if err != nil {
		return nil, err
	}

	l, err := s.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	return tls.NewListener(l, &tls.Config{
		GetCertificate: func(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return c.GetCertificate(hi)
		},
	}), nil
}

func (s *Service) GetDNSName(ctx context.Context) (string, error) {
	c, err := s.LocalClient()
	if err != nil {
		return "", err
	}

	status, err := waitUntilReady(ctx, c)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(status.Self.DNSName, "."), nil
}

func (s *Service) RedirectHTTP(ctx context.Context) error {
	c, err := s.LocalClient()
	if err != nil {
		return err
	}

	status, err := waitUntilReady(ctx, c)
	if err != nil {
		return err
	}

	l, err := s.Server.Listen("tcp", ":http")
	if err != nil {
		return err
	}
	defer l.Close()

	host, _, _ := strings.Cut(status.Self.DNSName, ".")
	fqdn := strings.TrimRight(status.Self.DNSName, ".")
	return http.Serve(
		l,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Host {
			case host, fqdn:
				u := *r.URL
				u.Host = fqdn
				u.Scheme = "https"
				http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
			default:
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		}))
}
