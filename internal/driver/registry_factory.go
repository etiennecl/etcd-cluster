package driver

import (
	"context"

	"github.com/clinia/x/logrusx"
	"github.com/etiennemtl/etcd-mini-cluster/internal/driver/config"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

func NewDefaultRegistry(ctx context.Context, flags *pflag.FlagSet) (Registry, error) {
	l := logrusx.New("Clinia Data Fabric", "0.0.0")
	c := config.New(ctx, l, nil)
	cp, err := config.NewProvider(ctx, flags, c)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize config provider")
	}
	c.WithSource(cp)

	r := &RegistryDefault{
		c: c,
		l: l,
	}

	if err := r.Init(ctx); err != nil {
		l.WithError(err).Error("Failed to initalize service registry.")
	}

	return r, nil
}
