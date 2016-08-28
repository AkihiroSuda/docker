package daemon

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/container"
	"github.com/docker/docker/daemon/cluster"
)

// RuntimeContext is introspection data for a container
type RuntimeContext struct {
	Container struct {
		ID     string
		Name   string
		Labels map[string]string
	}

	Task struct {
		ID     string
		Name   string
		Slot   int
		Labels map[string]string
	}

	Service struct {
		ID     string
		Name   string
		Labels map[string]string
	}

	Node struct {
		ID     string
		Name   string
		Labels map[string]string
	}
}

// introspectRuntimeContext returns RuntimeContext.
// Error is printed to logrus, and unknown field is set to an empty value.
func (daemon *Daemon) introspectRuntimeContext(c *container.Container) RuntimeContext {
	ctx := RuntimeContext{}
	ctx.Container.ID = c.ID
	ctx.Container.Name = c.Name
	ctx.Container.Labels = c.Config.Labels

	if cluster := daemon.GetCluster(); cluster != nil {
		introspectRuntimeContextTask(&ctx, c, cluster)
		introspectRuntimeContextService(&ctx, c, cluster)
		introspectRuntimeContextNode(&ctx, c, cluster)
	}
	return ctx
}

func introspectRuntimeContextTask(ctx *RuntimeContext, c *container.Container, cluster *cluster.Cluster) {
	taskID, ok := c.Config.Labels["com.docker.swarm.task.id"]
	if !ok {
		return
	}
	task, err := cluster.GetTask(taskID)
	if err != nil {
		logrus.Errorf("error while introspecting task %s: %v",
			taskID, err)
		return
	}
	ctx.Task.ID = task.ID
	ctx.Task.Name = task.Name
	ctx.Task.Slot = task.Slot
	ctx.Task.Labels = task.Labels
}

func introspectRuntimeContextService(ctx *RuntimeContext, c *container.Container, cluster *cluster.Cluster) {
	serviceID, ok := c.Config.Labels["com.docker.swarm.service.id"]
	if !ok {
		return
	}
	service, err := cluster.GetService(serviceID)
	if err != nil {
		logrus.Errorf("error while introspecting service %s: %v",
			serviceID, err)
		return
	}
	ctx.Service.ID = service.ID
	ctx.Service.Name = service.Spec.Name
	ctx.Service.Labels = service.Spec.Labels
}

func introspectRuntimeContextNode(ctx *RuntimeContext, c *container.Container, cluster *cluster.Cluster) {
	nodeID, ok := c.Config.Labels["com.docker.swarm.node.id"]
	if !ok {
		return
	}
	node, err := cluster.GetNode(nodeID)
	if err != nil {
		logrus.Errorf("error while introspecting node %s: %v",
			nodeID, err)
		return
	}
	ctx.Node.ID = node.ID
	ctx.Node.Name = node.Spec.Name
	ctx.Node.Labels = node.Spec.Labels
}
