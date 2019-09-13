// +build !linux

package daemon // import "github.com/docker/docker/daemon"

import (
	"context"
	"io"

	"github.com/containerd/containerd/containers"
	coci "github.com/containerd/containerd/oci"
	"github.com/docker/docker/container"
	"github.com/pkg/errors"
)

var supportsSeccomp = false

// WithSeccomp sets the seccomp profile
func WithSeccomp(daemon *Daemon, c *container.Container) coci.SpecOpts {
	return func(ctx context.Context, _ coci.Client, _ *containers.Container, s *coci.Spec) error {
		return nil
	}
}

// PrintDefaultSeccompProfile dumps the default profile to out
func PrintDefaultSeccompProfile(out io.Writer) error {
	return errors.New("seccomp is unsupported in this build")
}
