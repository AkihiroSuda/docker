// Package errorutils provides some utilities about errors
package errorutils

import (
	"errors"
)

var (
	ErrNotSwarmManager = errors.New("This node is not a swarm manager. Use \"docker swarm init\" or \"docker swarm join\" to connect this node to swarm and try again.")
)
