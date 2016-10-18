// +build !windows

package main

import (
	"strings"

	"github.com/docker/docker/pkg/integration/checker"
	"github.com/go-check/check"
)

func pruneNetworkAndVerify(c *check.C, kept, pruned []string) {
	_, _, err := dockerCmdWithError("network", "prune", "--force")
	c.Assert(err, checker.IsNil)
	out, _, err := dockerCmdWithError("network", "ls", "--format", "{{.Name}}")
	c.Assert(err, checker.IsNil)
	for _, s := range kept {
		c.Assert(out, checker.Contains, s)
	}
	for _, s := range pruned {
		c.Assert(out, checker.Not(checker.Contains), s)
	}
}

func (s *DockerNetworkSuite) TestPruneLocalNetwork(c *check.C) {
	dockerCmd(c, "network", "create", "nw-used") // used by container
	dockerCmd(c, "network", "create", "nw-unused")
	out, _ := dockerCmd(c, "run", "-d", "--net", "nw-used", "busybox", "top")
	containerID := strings.TrimSpace(out)
	waitRun(containerID)

	pruneNetworkAndVerify(c, []string{"nw-used"}, []string{"nw-unused"})
	dockerCmd(c, "rm", "-f", containerID)
	pruneNetworkAndVerify(c, []string{}, []string{"nw-used"})
}
