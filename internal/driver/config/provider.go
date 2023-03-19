package config

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/clinia/x/configx"
	"github.com/clinia/x/logrusx"
	"github.com/clinia/x/watcherx"
	"github.com/spf13/pflag"
)

type (
	Network struct {
		Host net.IP
	}
	Transport struct {
		Port int
	}
	Node struct {
		Name string
	}
	Cluster struct {
		Name               string
		InitialMasterNodes []string
	}
	Discovery struct {
		SeedHosts []string
	}
	Config struct {
		p   *configx.Provider
		l   *logrusx.Logger
		ctx context.Context
	}
	Provider interface {
		Config(ctx context.Context) *Config
	}
)

const (
	KeyNodeName                  = "node.name"
	KeyClusterName               = "cluster.name"
	KeyClusterInitialMasterNodes = "cluster.initial_master_nodes"
	KeyDiscoverySeedHosts        = "discovery.seed_hosts"
)

func New(ctx context.Context, l *logrusx.Logger, p *configx.Provider) *Config {
	return &Config{
		p:   p,
		l:   l,
		ctx: ctx,
	}
}

func NewDefault(ctx context.Context, flags *pflag.FlagSet, l *logrusx.Logger, opts ...configx.OptionModifier) (*Config, error) {
	c := New(ctx, l, nil)
	cp, err := NewProvider(ctx, flags, c, opts...)
	if err != nil {
		return nil, err
	}
	c.WithSource(cp)

	return c, nil
}

func NewProvider(ctx context.Context, flags *pflag.FlagSet, config *Config, opts ...configx.OptionModifier) (*configx.Provider, error) {
	p, err := configx.New(
		ctx,
		ConfigSchema,
		append(opts,
			configx.WithFlags(flags),
			configx.WithStderrValidationReporter(),
			configx.WithImmutables("serve"),
			configx.OmitKeysFromTracing(),
			configx.WithLogrusWatcher(config.l),
			configx.WithContext(ctx),
			configx.AttachWatcher(config.watcher),
		)...,
	)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (k *Config) Source() *configx.Provider {
	return k.p
}

func (k *Config) WithSource(p *configx.Provider) {
	k.p = p
	k.l.UseConfig(p)
}

func (k *Config) watcher(_ watcherx.Event, err error) {
	if err != nil {
		return
	}
}

func (k *Config) Node() *Node {
	return &Node{
		Name: k.p.String(KeyNodeName),
	}
}

func (k *Config) Cluster() *Cluster {
	return &Cluster{
		Name:               k.p.String(KeyClusterName),
		InitialMasterNodes: k.p.Strings(KeyClusterInitialMasterNodes),
	}
}

func (k *Config) Network() *Network {
	var ip net.IP
	v := k.p.StringF("network.host", "127.0.0.1")
	switch v {
	case "_local_":
		ip = net.ParseIP("127.0.0.1")
	case "_site_":
		ip = net.ParseIP("192.168.0.1")
	case "_global_":
		ip = net.ParseIP("8.8.8.8")
	default:
		ip = net.ParseIP(v)
	}

	return &Network{
		Host: ip,
	}
}

func (k *Config) Discovery() *Discovery {
	return &Discovery{
		SeedHosts: k.p.StringsF(KeyDiscoverySeedHosts, []string{
			"127.0.0.1",
		}),
	}
}

func (k *Config) Transport() *Transport {
	return &Transport{
		Port: k.p.IntF("transport.port", 9300),
	}
}

func (k *Config) TransportPeerURL() *url.URL {
	u, err := url.Parse(fmt.Sprintf("http://%s:%d", k.Network().Host, k.Transport().Port))
	if err != nil {
		panic(err)
	}
	return u
}

func (k *Config) TransportClientURL() *url.URL {
	u, err := url.Parse(fmt.Sprintf("http://%s:%d", k.Network().Host, k.Transport().Port+1))
	if err != nil {
		panic(err)
	}
	return u
}
