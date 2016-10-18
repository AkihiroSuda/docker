// +build !windows

package main

import (
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/integration/checker"
	"github.com/go-check/check"
)

func (s *DockerSwarmSuite) TestPruneNetwork(c *check.C) {
	d := s.AddDaemon(c, true, true)
	_, err := d.Cmd("network", "create", "n1") // used by container
	c.Assert(err, checker.IsNil)
	_, err = d.Cmd("network", "create", "n2")
	c.Assert(err, checker.IsNil)
	_, err = d.Cmd("network", "create", "n3", "--driver", "overlay") // used by service
	c.Assert(err, checker.IsNil)
	_, err = d.Cmd("network", "create", "n4", "--driver", "overlay")
	c.Assert(err, checker.IsNil)

	cName := "testprune"
	_, err = d.Cmd("run", "-d", "--name", cName, "--net", "n1", "busybox", "top")
	c.Assert(err, checker.IsNil)

	serviceName := "testprunesvc"
	replicas := 1
	out, err := d.Cmd("service", "create", "--name", serviceName,
		"--replicas", strconv.Itoa(replicas),
		"--network", "n3",
		"busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	// make sure task has been deployed.
	waitAndAssert(c, defaultReconciliationTimeout, d.checkActiveContainerCount, checker.Equals, replicas + 1)

	// prune and verify
	_, err = d.Cmd("network", "prune", "--force")
	c.Assert(err, checker.IsNil)
	out, err = d.Cmd("network", "ls", "--format", "{{.Name}}")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, "n1")
	c.Assert(out, checker.Not(checker.Contains), "n2")
	c.Assert(out, checker.Contains, "n3")
	c.Assert(out, checker.Not(checker.Contains), "n4")

	// remove containers
	_, err = d.Cmd("rm", cName)
	c.Assert(err, checker.IsNil)
	_, err = d.Cmd("service", "rm", serviceName)
	c.Assert(err, checker.IsNil)
	waitAndAssert(c, defaultReconciliationTimeout, d.checkActiveContainerCount, checker.Equals, 0)

	// prune and verify
	_, err = d.Cmd("network", "prune", "--force")
	c.Assert(err, checker.IsNil)
	out, err = d.Cmd("network", "ls", "--format", "{{.Name}}")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Not(checker.Contains), "n1")
	c.Assert(out, checker.Not(checker.Contains), "n3")
}
