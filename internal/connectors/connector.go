package connectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/vee-sh/veessh/internal/config"
)

type Connector interface {
	Name() string
	Exec(ctx context.Context, profile config.Profile, password string) error
}

var (
	registryMu sync.RWMutex
	registry   = map[config.Protocol]Connector{}
)

func Register(p config.Protocol, c Connector) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[p] = c
}

func Get(p config.Protocol) (Connector, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	c, ok := registry[p]
	if !ok {
		return nil, fmt.Errorf("no connector for protocol %s", p)
	}
	return c, nil
}
