package daemon

import (
	apitypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

// Cluster is the interface for github.com/docker/docker/daemon/cluster.(*Cluster).
type Cluster interface {
	GetNetwork(input string) (apitypes.NetworkResource, error)
	GetNetworks() ([]apitypes.NetworkResource, error)
	RemoveNetwork(input string) error
	// GetTask returns a task by an ID.
	GetTask(input string) (swarm.Task, error)
	// GetService returns a service based on an ID or name.
	GetService(input string) (swarm.Service, error)
	// GetNode returns a node based on an ID or name.
	GetNode(input string) (swarm.Node, error)
}
