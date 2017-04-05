package dockerfile

import (
	"fmt"
	"io"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/builder/dockerfile/dag"
	"github.com/docker/docker/builder/dockerfile/dag/scheduler"
	"github.com/docker/docker/builder/dockerfile/parallel"
)

// parallelBuilder is a parallel image builder
//  - Parses the Dockerfile, create stage DAG, and determine scheduling
//  - Calls NewBuilder() for each of stage in parallel, with Parallel=false, so as to ensure caches
//  - Return the result of building the last stage
type parallelBuilder struct {
	// common
	b      *Builder
	stdout io.Writer
	stderr io.Writer
	output io.Writer
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

	fmt.Fprintf(parb.stdout, "--> Prepared the experimental parallel builder.\n"+
		"  Total stages: %d\n"+
		"  DAG         : %+v\n"+
		"  Schedule    : %+v\n"+
		"Note that this feature is not stable yet. Use carefully.\n\n",
		len(parb.stages),
		parb.dag,
		parb.sched.String())
	return nil
}

func (parb *parallelBuilder) BuildStages() (map[int]string, error) {
	if err := parb.prepare(); err != nil {
		return nil, err
	}
	var mtx sync.Mutex
	imageIDs := make(map[int]string, 0)
	err := scheduler.ExecuteSchedule(parb.sched,
		int(parb.b.options.Parallelism),
		func(n dag.Node) error {
			imageID, err2 := parb.buildStage(int(n))
			if err2 != nil {
				return err2
			}
			mtx.Lock()
			imageIDs[int(n)] = imageID
			mtx.Unlock()
			return nil
		})
	return imageIDs, err
}

func cloneImageBuildOptionsForBuildingStage(c *types.ImageBuildOptions) *types.ImageBuildOptions {
	return &types.ImageBuildOptions{
		BuildArgs: c.BuildArgs,
		// TODO: Context?
		Parallel: false,
	}
}

func (parb *parallelBuilder) buildStage(idx int) (string, error) {
	return "", fmt.Errorf("unimplemented, while building stage %d", idx)
}
