package node

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/etiennemtl/etcd-mini-cluster/internal/driver/config"
	"github.com/etiennemtl/etcd-mini-cluster/internal/driver/logger"
	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
)

type (
	Node struct {
		Config *NodeConfig

		etcd         *embed.Etcd
		d            dependencies
		initialNodes []NodeConfig
	}
	NodeConfig struct {
		ID          string
		Name        string
		ClusterName string
		PeerURL     *url.URL
		ClientURL   *url.URL
		RootDir     string
	}
	dependencies interface {
		logger.Provider
		config.Provider
	}
)

func NewNode(ctx context.Context, d dependencies) *Node {
	return &Node{
		Config: &NodeConfig{
			Name:        d.Config(ctx).Node().Name,
			ClusterName: d.Config(ctx).Cluster().Name,
			RootDir:     fmt.Sprintf("data/%s", d.Config(ctx).Node().Name),
			PeerURL:     d.Config(ctx).TransportPeerURL(),
			ClientURL:   d.Config(ctx).TransportClientURL(),
		},
		d: d,
	}
}

func (n *Node) Bootstrap(ctx context.Context) error {
	discoveryNodes := n.Discover(ctx)

	endpoints := make([]string, len(discoveryNodes))
	for i, discoveryNode := range discoveryNodes {
		n.d.Logger().Infof("Found node: %s", discoveryNode.PeerURL.String())

		if n.Config.PeerURL.String() == discoveryNode.PeerURL.String() {
			return fmt.Errorf("node with transport %s already exists", n.Config.PeerURL.String())
		}

		endpoints[i] = discoveryNode.ClientURL.String()
	}

	if len(endpoints) == 0 {
		n.d.Logger().Infof("No nodes found, bootstrapping cluster %s", n.Config.ClusterName)
		return nil
	}

	// Create a temporary client to bootstrap the cluster
	client, err := clientv3.New(clientv3.Config{
		DialTimeout: 10 * time.Second,
		Endpoints:   endpoints,
	})

	if err != nil {
		return errors.WithStack(err)
	}
	defer client.Close()

	// Fetch all members to setup this node as a new member
	members, err := client.MemberList(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	nodes := make([]NodeConfig, len(members.Members))
	for i, member := range members.Members {
		peerUrl, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return errors.WithStack(err)
		}

		clientUrl, err := url.Parse(member.ClientURLs[0])
		if err != nil {
			return errors.WithStack(err)
		}

		nodes[i] = NodeConfig{
			ID:          fmt.Sprintf("%x", member.ID),
			Name:        member.Name,
			ClusterName: n.Config.ClusterName,
			PeerURL:     peerUrl,
			ClientURL:   clientUrl,
		}
	}

	_, err = client.MemberAdd(ctx, []string{n.Config.PeerURL.String()})
	if err != nil {
		return errors.WithStack(err)
	}

	n.initialNodes = nodes

	return nil
}

func (n *Node) Start(ctx context.Context, interruptCh <-chan interface{}) error {
	// Create the root directory if it does not exist
	err, _ := ensureDir(n.Config.RootDir)
	if err != nil {
		return errors.WithStack(err)
	}

	// Start the etcd server
	err = n.startEtcdServer(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	// Assign the node ID
	n.Config.ID = n.etcd.Server.ID().String()

	n.d.Logger().Infof("Cluster ID %s", n.etcd.Server.Cluster().ID())

	select {
	case err = <-n.etcd.Err():
		if err != nil {
			return err
		}
	case <-interruptCh:
		n.d.Logger().Infof("Shutdown signal received, stopping server...")

		ctx, cancel := context.WithCancel(ctx)
		n.Stop(ctx)
		cancel()
	}

	return nil
}

func (n *Node) Stop(ctx context.Context) error {
	_, err := n.etcd.Server.RemoveMember(ctx, uint64(n.etcd.Server.ID()))
	if err != nil {
		return errors.Wrap(err, "failed to remove node from cluster")
	}

	n.etcd.Server.Stop() // trigger a shutdown
	n.etcd.Close()
	return nil
}

// Discover returns a list of nodes that may already be in the cluster
func (n *Node) Discover(ctx context.Context) []NodeConfig {
	nodes := []NodeConfig{}

	// By defauly without additional configuration, a node will bind to the available loopback addresses
	// and scan ports 9300,9302,9304,9306,9308 for peer communication and 9301,9303,9305,9307,9309 for client communication.
	for _, u := range []struct{ peer, client int }{
		{9300, 9301},
		{9302, 9303},
		{9304, 9305},
		{9306, 9307},
		{9308, 9309},
	} {
		for _, h := range n.d.Config(ctx).Discovery().SeedHosts {
			// If the host is listening on the peer port and the client port, it is a node
			if hostListening(fmt.Sprintf("%s:%d", h, u.peer)) && hostListening(fmt.Sprintf("%s:%d", h, u.client)) {
				peerUrl, err := url.Parse(fmt.Sprintf("http://%s:%d", h, u.peer))
				if err != nil {
					continue
				}

				clientUrl, err := url.Parse(fmt.Sprintf("http://%s:%d", h, u.client))
				if err != nil {
					continue
				}

				nodes = append(nodes, NodeConfig{
					PeerURL:   peerUrl,
					ClientURL: clientUrl,
				})
			}
		}
	}

	return nodes
}

func (n *Node) ClusterID() string {
	return n.etcd.Server.Cluster().ID().String()
}

func (n *Node) startEtcdServer(ctx context.Context) error {
	// Create the etcd configuration
	etcdConfig := n.etcdConfig()

	// Start the etcd server
	etcd, err := embed.StartEtcd(etcdConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	// Wait for the etcd server to be ready
	select {
	case <-etcd.Server.ReadyNotify():
		log.Printf("Server %s is ready", etcd.Server.ID())
	case <-time.After(60 * time.Second):
		etcd.Server.Stop() // trigger a shutdown
		etcd.Close()
		log.Printf("Server %s took too long to start", etcd.Server.ID())
		return errors.New("Etcd Server took too long to start!")
	}

	n.etcd = etcd

	return nil
}

func (n *Node) etcdConfig() *embed.Config {
	etcdCfg := embed.NewConfig()
	etcdCfg.Name = n.Config.Name
	etcdCfg.Dir = n.Config.RootDir

	// Node etcd urls
	etcdCfg.LPUrls = []url.URL{*n.Config.PeerURL}   // Listen Peer Urls
	etcdCfg.LCUrls = []url.URL{*n.Config.ClientURL} // Listen Client Urls
	etcdCfg.APUrls = []url.URL{*n.Config.PeerURL}   // Advertise Peer Urls
	etcdCfg.ACUrls = []url.URL{*n.Config.ClientURL} // Advertise Client Urls

	// Initial Cluster
	nodeClusterUrls := []string{
		clusterUrl(n.Config.Name, n.Config.PeerURL.String()),
	}
	for _, node := range n.initialNodes {
		nodeClusterUrls = append(nodeClusterUrls, clusterUrl(node.Name, node.PeerURL.String()))
	}
	etcdCfg.InitialCluster = strings.Join(nodeClusterUrls, ",")

	// Initial Cluster Token (Cluster Name)
	etcdCfg.InitialClusterToken = n.d.Config(context.Background()).Cluster().Name

	// Initial Cluster State
	if len(n.initialNodes) == 0 {
		etcdCfg.ClusterState = "new"
	} else {
		etcdCfg.ClusterState = "existing"
	}

	return etcdCfg
}

func hostListening(host string) bool {
	ln, err := net.Listen("tcp", host)
	if err != nil {
		return true
	}
	ln.Close()
	return false
}

func ensureDir(path string) (error, bool) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// TODO: modify persmissions
		err := os.Mkdir(path, 0755)
		if err != nil {
			return errors.WithStack(err), false
		}
	}

	return nil, true
}

func clusterUrl(name string, url string) string {
	return fmt.Sprintf("%s=%s", name, url)
}
