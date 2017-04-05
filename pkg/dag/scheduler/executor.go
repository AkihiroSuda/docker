package scheduler

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/dag"
	"golang.org/x/sync/errgroup"
)

func executeSchedule(sched *Schedule, exe func(dag.Node) error) error {
	if err := exe(sched.Node); err != nil {
		return err
	}
	var (
		eg errgroup.Group
	)
	for _, c := range sched.Children {
		c := c
		eg.Go(func() error {
			return executeSchedule(c, exe)
		})
	}
	return eg.Wait()
}

func ExecuteSchedule(root *ScheduleRoot, parallelism int, exe func(dag.Node) error) error {
	if parallelism == 0 {
		parallelism = countNodes(root)
	}
	logrus.Warnf("parallelism (currently ignored) : %d", parallelism)
	var (
		eg errgroup.Group
	)
	for _, c := range root.Children {
		c := c
		eg.Go(func() error {
			return executeSchedule(c, exe)
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

func countNodes(root *ScheduleRoot) int {
	m := make(map[dag.Node]struct{}, 0)
	for _, c := range root.Children {
		_countNodes(m, c)
	}
	return len(m)
}
