package dockerfile

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/builder/dockerfile/dag"
	"github.com/docker/docker/builder/dockerfile/dag/scheduler"
	"github.com/docker/docker/builder/dockerfile/parallel"
)

func cloneImageBuildOptionsForCachingStage(c *types.ImageBuildOptions) *types.ImageBuildOptions {
	return &types.ImageBuildOptions{
		BuildArgs: c.BuildArgs,
		// TODO: Context?
		Parallel: false,
	}
}

// parallelBuilder is tricky
//  - Parses the Dockerfile, create stage DAG, and determine scheduling
//  - Calls NewBuilder() for each of stage in parallel, with Parallel=false, so as to ensure caches
//  - Return the result of building the last stage
type parallelBuilder struct {
	// common
	b *Builder
	// specific to parallel builder
	stages []*parallel.Stage
	dag    *dag.Graph
	sched  *scheduler.ScheduleRoot
}

func (parb *parallelBuilder) prepare() error {
	var err error
	parb.stages, err = parallel.ParseStages(parb.b.dockerfile)
	if err != nil {
		return err
	}
	logrus.Debugf("[PARALLEL BUILDER] Detected %d build stages", len(parb.stages))
	parb.dag, err = parallel.CreateDAG(parb.stages)
	if err != nil {
		return err
	}
	logrus.Debugf("[PARALLEL BUILDER] DAG: %+v", parb.dag)
	parb.sched = scheduler.DetermineSchedule(parb.dag)
	logrus.Debugf("[PARALLEL BUILDER] Schedule: %s", parb.sched.String())
	return nil
}

func (parb *parallelBuilder) BuildStages() ([]string, error) {
	if err := parb.prepare(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
