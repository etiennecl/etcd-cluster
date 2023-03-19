# Etcd Cluster

This projects provides example code to create an embedded etcd cluster in a Golang server.

The application can than use the cluster with a central key value storage acting as a shared distributed cluster state without depending on any extra database.

# Cluster

Every instance of a server, is treated as a node inside cluster. Each node have a pair of 
peer url and client url for internal communication between nodes. The `transport.port` configuration can be used to set this value of the peer port. By default it will use `2300`. The client url is automatically configured by adding 1 to the peer url. Therefore when configuring multiple nodes in a cluster make sure to reserve 2 consecutive ports for each node.

Each instance of a server, require `cluster.name` and `node.name` to be set in order for the cluster to be configured accordingly and the node to join the desired cluster.

**Discovery**

By default, without additional configuration, a server will listen to all loopback addresses and ports `2300-2309` in order to find any node/cluster already running and join it if so.

In a use case where a 3 nodes cluster need to be boostrap from scratch, `cluster.initial_master_nodes` which correspond to all `node.name` that will be part of the cluster need to be specified.

If nodes are not port of the same network/host the `discovery.seed_hosts` need to be specifided in order for the cluster to listen to those IP addresses.

## TODO

- Handle Node lifecycle:
  - Join
  - Remove
- Cluster teardown
- Cluster Storage Interface