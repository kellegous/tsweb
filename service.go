package tsweb

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/multierr"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

type Service struct {
	Server      *tsnet.Server
	Listener    net.Listener
	StateDir    string
	LocalClient *tailscale.LocalClient
}

func (s *Service) Close() error {
	var res error
	if err := s.Listener.Close(); err != nil {
		res = multierr.Append(res, err)
	}
	if err := s.Server.Close(); err != nil {
		res = multierr.Append(res, err)
	}
	return res
}

func (s *Service) RedirectHTTP(ctx context.Context) error {
	status, err := s.LocalClient.Status(ctx)
	if err != nil {
		return err
	}

	l, err := s.Server.Listen("tcp", ":80")
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

func Start(
	ctx context.Context,
	opts *ServiceOptions,
) (*Service, error) {
	s := &tsnet.Server{
		Dir:      opts.StateDir,
		Hostname: opts.Hostname,
		AuthKey:  opts.AuthKey,
		Logf:     opts.Logger,
	}

	lc, err := s.LocalClient()
	if err != nil {
		return nil, err
	}

	if _, err := waitUntilRunning(ctx, lc); err != nil {
		return nil, err
	}

	l, err := s.Listen("tcp", ":443")
	if err != nil {
		return nil, err
	}

	return &Service{
		Server: s,
		Listener: tls.NewListener(l, &tls.Config{
			GetCertificate: func(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return lc.GetCertificate(hi)
			},
		}),
		StateDir:    opts.StateDir,
		LocalClient: lc,
	}, nil
}

func waitUntilRunning(
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
