package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/docker/pkg/integration/checker"
	"github.com/go-check/check"
)

func skipUntil28527(c *check.c) {
	c.Skip("This test does not work until #28527 gets implemented (--volume type=TYPE,...)")
}

func (s *DockerSuite) TestRunIntrospection(c *check.C) {
	testRequires(c, DaemonIsLinux, ExperimentalDaemon)
	skipUntil28527(c)
	cName := "test-run-introspection"
	mount := "/foo"
	delim := "\n"
	out, _ := dockerCmd(c,
		"run",
		"-d",
		"--name", cName,
		"--volume", "type=introspection,target="+mount,
		"busybox", "top")
	cID := strings.TrimSpace(out)
	c.Assert(waitRun(cName), check.IsNil)

	out, _ = dockerCmd(c, "exec", cName, "ls", mount)
	dirs := strings.Split(strings.TrimSpace(out), "\n")
	sort.Strings(dirs)
	c.Assert(dirs, check.DeepEquals, []string{"container", "daemon"})

	out, _ = dockerCmd(c, "exec", cName, "cat", filepath.Join(mount, "container", "id"))
	c.Assert(out, check.Equals, cID+delim)
	out, _ = dockerCmd(c, "exec", cName, "cat", filepath.Join(mount, "container", "name"))
	c.Assert(out, check.Equals, cName+delim)
	out, _ = dockerCmd(c, "exec", cName, "cat", filepath.Join(mount, "container", "fullname"))
	c.Assert(out, check.Equals, "/"+cName+delim)
	out, _ = dockerCmd(c, "exec", cName, "cat", filepath.Join(mount, "daemon", "name"))
	hostname, _ := os.Hostname()
	c.Assert(out, check.Equals, hostname+delim)

	dockerCmd(c, "rm", "-f", cName)
}

func (s *DockerSuite) TestRunIntrospectionError(c *check.C) {
	testRequires(c, DaemonIsLinux, ExperimentalDaemon)
	skipUntil28527(c)
	type testCase struct {
		mountOpts     []string
		expectedError string
	}
	cases := []testCase{
		{
			mountOpts:     []string{"--volume", "type=introspection"},
			expectedError: "target is required",
		},
		{
			mountOpts:     []string{"--volume", "type=introspection,target=/foo,readonly=false"},
			expectedError: "cannot set readonly=false explicitly",
		},
		{
			mountOpts:     []string{"--volume", "type=introspection,target=/foo,source=/bar"},
			expectedError: "Source must not be specified",
		},
	}
	for _, tc := range cases {
		out, _, err := dockerCmdWithError(append([]string{"run", "-d"}, append(tc.mountOpts, "busybox", "top")...)...)
		c.Assert(err, checker.NotNil)
		c.Assert(out, checker.Contains, tc.expectedError)
	}
}

func (s *DockerSuite) TestRunIntrospectionNonExperimental(c *check.C) {
	testRequires(c, DaemonIsLinux, NotExperimentalDaemon)
	skipUntil28527(c)
	out, _, err := dockerCmdWithError("run", "-d", "--volume", "type=introspection,target=/foo", "busybox", "top")
	c.Assert(err, checker.NotNil)
	c.Assert(out, checker.Contains, "introspection mount is only supported in experimental mode")
}
