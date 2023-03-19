package driver

import (
	"context"

	"github.com/clinia/x/logrusx"
	"github.com/etiennemtl/etcd-mini-cluster/internal/driver/config"
)

var (
	_ Registry = (*RegistryDefault)(nil)
)

type RegistryDefault struct {
	l *logrusx.Logger
	c *config.Config
}

// Logger implements Registry
func (r *RegistryDefault) Logger() *logrusx.Logger {
	if r.l == nil {
		r.l = logrusx.New("Clinia Data Fabric", "")
	}
	return r.l
}

// Init implements Registry
func (r *RegistryDefault) Init(ctx context.Context) error {
	return nil
}

// Config implements Registry
func (r *RegistryDefault) Config(ctx context.Context) *config.Config {
	if r.c == nil {
		panic("configuration not set")
	}

	return r.c
}

// Close implements Registry
func (*RegistryDefault) Close() error {
	return nil
}
