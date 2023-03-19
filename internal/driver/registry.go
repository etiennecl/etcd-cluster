package driver

import (
	"context"

	"github.com/etiennemtl/etcd-mini-cluster/internal/driver/config"
	"github.com/etiennemtl/etcd-mini-cluster/internal/driver/logger"
)

type (
	Registry interface {
		config.Provider
		logger.Provider

		Init(ctx context.Context) error
		Close() error
	}
)
