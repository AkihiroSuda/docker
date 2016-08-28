// +build !experimental

package daemon

import (
	"errors"

	"github.com/docker/docker/container"
)

func (daemon *Daemon) updateIntrospection(c *container.Container, opts introspectionOptions) error {
	return errors.New("introspection requires an experimental build")
}
