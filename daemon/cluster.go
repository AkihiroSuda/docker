package daemon

import (
	types "github.com/docker/docker/api/types/swarm"
)

// IntrospectableCluster is the interface for github.com/docker/docker/daemon/cluster.(*Cluster).
type IntrospectableCluster interface {
	// GetTask returns a task by an ID.
	GetTask(input string) (types.Task, error)
	// GetService returns a service based on an ID or name.
	GetService(input string) (types.Service, error)
	// GetNode returns a node based on an ID or name.
	GetNode(input string) (types.Node, error)
}
