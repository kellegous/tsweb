package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kellegous/tsweb"
	"tailscale.com/tsnet"
)

type Flags struct {
	AuthKey  string
	Hostname string
	StateDir string
}

func (f *Flags) Register(fs *flag.FlagSet) {
	fs.StringVar(
		&f.AuthKey,
		"auth-key",
		"",
		"the tailscale auth key for your tailnet")
	fs.StringVar(
		&f.Hostname,
		"hostname",
		"example",
		"hostname for the virtual service")
	fs.StringVar(
		&f.StateDir,
		"state-dir",
		"tsweb",
		"the directory where tailscale state will be stored")
}

func main() {
	var flags Flags
	flags.Register(flag.CommandLine)
	flag.Parse()

	// ensure we have the state directory
	os.MkdirAll(flags.StateDir, 0700)

	ctx := context.Background()

	s, err := tsweb.Start(&tsnet.Server{
		AuthKey:  flags.AuthKey,
		Hostname: flags.Hostname,
		Dir:      flags.StateDir,
	})
	if err != nil {
		log.Panic(err)
	}
	defer s.Close()

	ech := make(chan error)
	go func() {
		ech <- s.RedirectHTTP(ctx)
	}()

	l, err := s.ListenTLS("tcp", ":https")
	if err != nil {
		log.Panic(err)
	}

	c, err := s.LocalClient()
	if err != nil {
		log.Panic(err)
	}

	go func() {
		ech <- http.Serve(l, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				who, err := c.WhoIs(r.Context(), r.RemoteAddr)
				if err != nil {
					log.Panic(err)
				}

				w.Header().Set("Content-Type", "text/plain;charset=utf8")
				fmt.Fprintf(w, "Hello %s", who.Node.Name)
			},
		))
	}()

	log.Panic(<-ech)
}
