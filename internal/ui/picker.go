package ui

import "github.com/vee-sh/veessh/internal/config"

func effectivePort(p config.Profile) int {
	if p.Port > 0 {
		return p.Port
	}
	switch p.Protocol {
	case config.ProtocolSSH, config.ProtocolSFTP:
		return 22
	case config.ProtocolTelnet:
		return 23
	default:
		return 0
	}
}
