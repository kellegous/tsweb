package tsweb

import "tailscale.com/types/logger"

type ServiceOptions struct {
	AuthKey  string
	StateDir string
	Hostname string
	Logger   logger.Logf
}
