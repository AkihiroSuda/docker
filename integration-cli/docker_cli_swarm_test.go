// +build !windows

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/integration/checker"
	"github.com/go-check/check"
)

func (s *DockerSwarmSuite) TestSwarmUpdate(c *check.C) {
	d := s.AddDaemon(c, true, true)

	getSpec := func() swarm.Spec {
		sw := d.getSwarm(c)
		return sw.Spec
	}

	out, err := d.Cmd("swarm", "update", "--cert-expiry", "30h", "--dispatcher-heartbeat", "11s")
	c.Assert(err, checker.IsNil, check.Commentf("out: %v", out))

	spec := getSpec()
	c.Assert(spec.CAConfig.NodeCertExpiry, checker.Equals, 30*time.Hour)
	c.Assert(spec.Dispatcher.HeartbeatPeriod, checker.Equals, 11*time.Second)

	// setting anything under 30m for cert-expiry is not allowed
	out, err = d.Cmd("swarm", "update", "--cert-expiry", "15m")
	c.Assert(err, checker.NotNil)
	c.Assert(out, checker.Contains, "minimum certificate expiry time")
	spec = getSpec()
	c.Assert(spec.CAConfig.NodeCertExpiry, checker.Equals, 30*time.Hour)
}

func (s *DockerSwarmSuite) TestSwarmInit(c *check.C) {
	d := s.AddDaemon(c, false, false)

	getSpec := func() swarm.Spec {
		sw := d.getSwarm(c)
		return sw.Spec
	}

	out, err := d.Cmd("swarm", "init", "--cert-expiry", "30h", "--dispatcher-heartbeat", "11s")
	c.Assert(err, checker.IsNil, check.Commentf("out: %v", out))

	spec := getSpec()
	c.Assert(spec.CAConfig.NodeCertExpiry, checker.Equals, 30*time.Hour)
	c.Assert(spec.Dispatcher.HeartbeatPeriod, checker.Equals, 11*time.Second)

	c.Assert(d.Leave(true), checker.IsNil)
	time.Sleep(500 * time.Millisecond) // https://github.com/docker/swarmkit/issues/1421
	out, err = d.Cmd("swarm", "init")
	c.Assert(err, checker.IsNil, check.Commentf("out: %v", out))

	spec = getSpec()
	c.Assert(spec.CAConfig.NodeCertExpiry, checker.Equals, 90*24*time.Hour)
	c.Assert(spec.Dispatcher.HeartbeatPeriod, checker.Equals, 5*time.Second)
}

func (s *DockerSwarmSuite) TestSwarmInitIPv6(c *check.C) {
	testRequires(c, IPv6)
	d1 := s.AddDaemon(c, false, false)
	out, err := d1.Cmd("swarm", "init", "--listen-addr", "::1")
	c.Assert(err, checker.IsNil, check.Commentf("out: %v", out))

	d2 := s.AddDaemon(c, false, false)
	out, err = d2.Cmd("swarm", "join", "::1")
	c.Assert(err, checker.IsNil, check.Commentf("out: %v", out))

	out, err = d2.Cmd("info")
	c.Assert(err, checker.IsNil, check.Commentf("out: %v", out))
	c.Assert(out, checker.Contains, "Swarm: active")
}

func (s *DockerSwarmSuite) TestSwarmIncompatibleDaemon(c *check.C) {
	// init swarm mode and stop a daemon
	d := s.AddDaemon(c, true, true)
	info, err := d.info()
	c.Assert(err, checker.IsNil)
	c.Assert(info.LocalNodeState, checker.Equals, swarm.LocalNodeStateActive)
	c.Assert(d.Stop(), checker.IsNil)

	// start a daemon with --cluster-store and --cluster-advertise
	err = d.Start("--cluster-store=consul://consuladdr:consulport/some/path", "--cluster-advertise=1.1.1.1:2375")
	c.Assert(err, checker.NotNil)
	content, _ := ioutil.ReadFile(d.logFile.Name())
	c.Assert(string(content), checker.Contains, "--cluster-store and --cluster-advertise daemon configurations are incompatible with swarm mode")

	// start a daemon with --live-restore
	err = d.Start("--live-restore")
	c.Assert(err, checker.NotNil)
	content, _ = ioutil.ReadFile(d.logFile.Name())
	c.Assert(string(content), checker.Contains, "--live-restore daemon configuration is incompatible with swarm mode")
	// restart for teardown
	c.Assert(d.Start(), checker.IsNil)
}

// Test case for #24090
func (s *DockerSwarmSuite) TestSwarmNodeListHostname(c *check.C) {
	d := s.AddDaemon(c, true, true)

	// The first line should contain "HOSTNAME"
	out, err := d.Cmd("node", "ls")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.Split(out, "\n")[0], checker.Contains, "HOSTNAME")
}

// Test case for #24270
func (s *DockerSwarmSuite) TestSwarmServiceListFilter(c *check.C) {
	d := s.AddDaemon(c, true, true)

	name1 := "redis-cluster-md5"
	name2 := "redis-cluster"
	name3 := "other-cluster"
	out, err := d.Cmd("service", "create", "--name", name1, "busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	out, err = d.Cmd("service", "create", "--name", name2, "busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	out, err = d.Cmd("service", "create", "--name", name3, "busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	filter1 := "name=redis-cluster-md5"
	filter2 := "name=redis-cluster"

	// We search checker.Contains with `name+" "` to prevent prefix only.
	out, err = d.Cmd("service", "ls", "--filter", filter1)
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, name1+" ")
	c.Assert(out, checker.Not(checker.Contains), name2+" ")
	c.Assert(out, checker.Not(checker.Contains), name3+" ")

	out, err = d.Cmd("service", "ls", "--filter", filter2)
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, name1+" ")
	c.Assert(out, checker.Contains, name2+" ")
	c.Assert(out, checker.Not(checker.Contains), name3+" ")

	out, err = d.Cmd("service", "ls")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, name1+" ")
	c.Assert(out, checker.Contains, name2+" ")
	c.Assert(out, checker.Contains, name3+" ")
}

func (s *DockerSwarmSuite) TestSwarmNodeListFilter(c *check.C) {
	d := s.AddDaemon(c, true, true)

	out, err := d.Cmd("node", "inspect", "--format", "{{ .Description.Hostname }}", "self")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")
	name := strings.TrimSpace(out)

	filter := "name=" + name[:4]

	out, err = d.Cmd("node", "ls", "--filter", filter)
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, name)

	out, err = d.Cmd("node", "ls", "--filter", "name=none")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Not(checker.Contains), name)
}

func (s *DockerSwarmSuite) TestSwarmNodeTaskListFilter(c *check.C) {
	d := s.AddDaemon(c, true, true)

	name := "redis-cluster-md5"
	out, err := d.Cmd("service", "create", "--name", name, "--replicas=3", "busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	// make sure task has been deployed.
	waitAndAssert(c, defaultReconciliationTimeout, d.checkActiveContainerCount, checker.Equals, 3)

	filter := "name=redis-cluster"

	out, err = d.Cmd("node", "ps", "--filter", filter, "self")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Contains, name+".1")
	c.Assert(out, checker.Contains, name+".2")
	c.Assert(out, checker.Contains, name+".3")

	out, err = d.Cmd("node", "ps", "--filter", "name=none", "self")
	c.Assert(err, checker.IsNil)
	c.Assert(out, checker.Not(checker.Contains), name+".1")
	c.Assert(out, checker.Not(checker.Contains), name+".2")
	c.Assert(out, checker.Not(checker.Contains), name+".3")
}

// Test case for #25375
func (s *DockerSwarmSuite) TestSwarmPublishAdd(c *check.C) {
	d := s.AddDaemon(c, true, true)

	name := "top"
	out, err := d.Cmd("service", "create", "--name", name, "--label", "x=y", "busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	out, err = d.Cmd("service", "update", "--publish-add", "80:80", name)
	c.Assert(err, checker.IsNil)

	out, err = d.cmdRetryOutOfSequence("service", "update", "--publish-add", "80:80", name)
	c.Assert(err, checker.IsNil)

	out, err = d.cmdRetryOutOfSequence("service", "update", "--publish-add", "80:80", "--publish-add", "80:20", name)
	c.Assert(err, checker.NotNil)

	out, err = d.cmdRetryOutOfSequence("service", "update", "--publish-add", "80:20", name)
	c.Assert(err, checker.IsNil)

	out, err = d.Cmd("service", "inspect", "--format", "{{ .Spec.EndpointSpec.Ports }}", name)
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Equals, "[{ tcp 20 80}]")
}

func (s *DockerSwarmSuite) TestSwarmServiceWithGroup(c *check.C) {
	d := s.AddDaemon(c, true, true)

	name := "top"
	out, err := d.Cmd("service", "create", "--name", name, "--user", "root:root", "--group-add", "wheel", "--group-add", "audio", "--group-add", "staff", "--group-add", "777", "busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	// make sure task has been deployed.
	waitAndAssert(c, defaultReconciliationTimeout, d.checkActiveContainerCount, checker.Equals, 1)

	out, err = d.Cmd("ps", "-q")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	container := strings.TrimSpace(out)

	out, err = d.Cmd("exec", container, "id")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Equals, "uid=0(root) gid=0(root) groups=10(wheel),29(audio),50(staff),777")
}

func (s *DockerSwarmSuite) TestSwarmContainerAutoStart(c *check.C) {
	d := s.AddDaemon(c, true, true)

	out, err := d.Cmd("network", "create", "--attachable", "-d", "overlay", "foo")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	out, err = d.Cmd("run", "-id", "--restart=always", "--net=foo", "--name=test", "busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	out, err = d.Cmd("ps", "-q")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	d.Restart()

	out, err = d.Cmd("ps", "-q")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")
}

func introspectSwarm(c *check.C, d *SwarmDaemon, containerID string) (*types.ContainerJSON, *swarm.Task, *swarm.Service, *swarm.Node) {
	out, err := d.Cmd("inspect", containerID, "--format", "{{json .}}")
	c.Assert(err, checker.IsNil)
	var container types.ContainerJSON
	c.Assert(json.Unmarshal([]byte(out), &container), checker.IsNil)
	task := d.getTask(c, container.Config.Labels["com.docker.swarm.task.id"])
	service := d.getService(c, container.Config.Labels["com.docker.swarm.service.id"])
	node := d.getNode(c, container.Config.Labels["com.docker.swarm.node.id"])
	return &container, &task, service, node
}

func testIntrospectionDaemon(c *check.C, d *Daemon, container *types.ContainerJSON, mountpoint string) {
	nameFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "daemon/name"))
	c.Assert(err, checker.IsNil)
	hostname, _ := os.Hostname()
	c.Assert(nameFile, checker.Equals, hostname+"\n")
}

func testIntrospectionContainer(c *check.C, d *Daemon, container *types.ContainerJSON, mountpoint string) {
	idFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "container/id"))
	c.Assert(err, checker.IsNil)
	c.Assert(idFile, checker.Equals, container.ID+"\n")

	nameFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "container/name"))
	c.Assert(err, checker.IsNil)
	fullnameFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "container/fullname"))
	c.Assert(err, checker.IsNil)
	c.Assert(fullnameFile, checker.Equals, "/"+nameFile)
	for k, v := range container.Config.Labels {
		f, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "container/labels", k))
		c.Assert(err, checker.IsNil)
		if v == "" {
			c.Assert(f, checker.Equals, "")
		} else {
			c.Assert(f, checker.Equals, v+"\n")
		}
	}
}

func testIntrospectionTask(c *check.C, d *SwarmDaemon, container *types.ContainerJSON, task *swarm.Task, mountpoint string) {
	idFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "task/id"))
	c.Assert(err, checker.IsNil)
	c.Assert(idFile, checker.Equals, task.ID+"\n")

	nameFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "task/name"))
	c.Assert(err, checker.IsNil)
	c.Assert(nameFile, checker.Equals, container.Config.Labels["com.docker.swarm.task.name"]+"\n") // not *swarm.Task.Spec.Name

	slotFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "task/slot"))
	c.Assert(err, checker.IsNil)
	c.Assert(slotFile, checker.Equals, fmt.Sprintf("%d\n", task.Slot))
	// no labels yet
}

func testIntrospectionService(c *check.C, d *SwarmDaemon, container *types.ContainerJSON, service *swarm.Service, mountpoint string) {
	idFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "service/id"))
	c.Assert(err, checker.IsNil)
	c.Assert(idFile, checker.Equals, service.ID+"\n")

	nameFile, err := d.Cmd("exec", container.ID, "cat", filepath.Join(mountpoint, "service/name"))
	c.Assert(err, checker.IsNil)
	c.Assert(nameFile, checker.Equals, service.Spec.Name+"\n")
	// no labels yet
}

func (s *DockerSwarmSuite) TestSwarmServiceWithIntrospectionVolume(c *check.C) {
	d := s.AddDaemon(c, true, true)
	name := "introspection"
	replicas := 3
	mountpoint := "/foo"
	out, err := d.Cmd("service", "create", "--name", name,
		"--replicas", strconv.Itoa(replicas),
		"--mount", "type=introspection,dst="+mountpoint,
		"busybox", "top")
	c.Assert(err, checker.IsNil)
	c.Assert(strings.TrimSpace(out), checker.Not(checker.Equals), "")

	// make sure task has been deployed.
	waitAndAssert(c, defaultReconciliationTimeout, d.checkActiveContainerCount, checker.Equals, replicas)

	out, err = d.Cmd("ps", "-q", "--no-trunc")
	c.Assert(err, checker.IsNil)
	containers := strings.Split(strings.TrimSpace(out), "\n")
	c.Assert(containers, checker.HasLen, replicas)

	for _, containerID := range containers {
		container, task, service, _ := introspectSwarm(c, d, containerID)
		testIntrospectionDaemon(c, d.Daemon, container, mountpoint)
		testIntrospectionContainer(c, d.Daemon, container, mountpoint)
		testIntrospectionTask(c, d, container, task, mountpoint)
		testIntrospectionService(c, d, container, service, mountpoint)
	}
}
