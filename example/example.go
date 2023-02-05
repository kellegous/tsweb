package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kellegous/tsweb"
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

	s, err := tsweb.Start(
		ctx,
		&tsweb.ServiceOptions{
			AuthKey:  flags.AuthKey,
			Hostname: flags.Hostname,
			StateDir: flags.StateDir,
		})
	if err != nil {
		log.Panic(err)
	}
	defer s.Close()

	ech := make(chan error)
	go func() {
		ech <- s.RedirectHTTP(ctx)
	}()

	go func() {
		ech <- http.Serve(s.Listener, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				who, err := s.LocalClient.WhoIs(ctx, r.RemoteAddr)
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
