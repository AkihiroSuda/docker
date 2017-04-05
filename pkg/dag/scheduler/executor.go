package scheduler

import (
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/dag"
	"golang.org/x/sync/errgroup"
)

type executor struct {
	onces map[dag.Node]*sync.Once
}

func (x *executor) executeSchedule(sched *Schedule, exe func(dag.Node) error) error {
	var err error
	x.onces[sched.Node].Do(func() {
		err = exe(sched.Node)
	})
	if err != nil {
		return err
	}
	var (
		eg errgroup.Group
	)
	for _, c := range sched.Children {
		c := c
		eg.Go(func() error {
			return x.executeSchedule(c, exe)
		})
	}
	return eg.Wait()
}

func ExecuteSchedule(root *ScheduleRoot, parallelism int, exe func(dag.Node) error) error {
	nodes := countNodes(root)
	if parallelism == 0 {
		parallelism = len(nodes)
	}
	logrus.Warnf("parallelism (currently ignored) : %d", parallelism)
	x := &executor{
		onces: make(map[dag.Node]*sync.Once, len(nodes)),
	}
	for _, n := range nodes {
		x.onces[n] = new(sync.Once)
	}
	var (
		eg errgroup.Group
	)
	for _, c := range root.Children {
		c := c
		eg.Go(func() error {
			return x.executeSchedule(c, exe)
		})
	}
	return eg.Wait()
}

func _countNodes(m map[dag.Node]struct{}, s *Schedule) {
	m[s.Node] = struct{}{}
	for _, c := range s.Children {
		_countNodes(m, c)
	}
}

func countNodes(root *ScheduleRoot) []dag.Node {
	m := make(map[dag.Node]struct{}, 0)
	for _, c := range root.Children {
		_countNodes(m, c)
	}
	var res []dag.Node
	for n := range m {
		res = append(res, n)
	}
	return res
}
