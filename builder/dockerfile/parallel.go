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
//  - Return the result of the last stage
type parallelBuilder struct {
	b      *Builder
	stdout io.Writer
	stderr io.Writer
	output io.Writer
}

func (parb *parallelBuilder) prepare() ([]*parallel.Stage, *scheduler.ScheduleRoot, error) {
	stages, err := parallel.ParseStages(parb.b.dockerfile)
	if err != nil {
		return nil, nil, err
	}
	logrus.Debugf("[PARALLEL BUILDER] Detected %d build stages", len(stages))
	daggy, err := parallel.CreateDAG(stages)
	if err != nil {
		return nil, nil, err
	}
	logrus.Debugf("[PARALLEL BUILDER] DAG: %+v", daggy)
	sched := scheduler.DetermineSchedule(daggy)
	logrus.Debugf("[PARALLEL BUILDER] Schedule: %s", sched.String())

	fmt.Fprintf(parb.stdout, "Prepared the experimental parallel builder.\n"+
		"  Total stages: %d\n"+
		"  DAG         : %+v\n"+
		"  Schedule    : %+v\n"+
		"Note that this feature is not stable yet. Use carefully.\n\n",
		len(stages),
		daggy,
		sched.String())
	return stages, sched, nil
}

func (parb *parallelBuilder) BuildStages() (map[int]string, error) {
	stages, sched, err := parb.prepare()
	if err != nil {
		return nil, err
	}
	var mtx sync.Mutex
	imageIDs := make(map[int]string, 0)
	err = scheduler.ExecuteSchedule(sched,
		int(parb.b.options.Parallelism),
		func(n dag.Node) error {
			imageID, err2 := parb.buildStage(stages, int(n))
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

func (parb *parallelBuilder) buildStage(stages []*parallel.Stage, idx int) (string, error) {
	logrus.Debugf("[PARALLEL BUILDER] Building stage %d", idx)
	fmt.Fprintf(parb.stdout, "Building stage %d\n", idx)
	df, err := parallel.CreateDockerfileThatContainsDependencyStages(stages, idx)
	if err != nil {
		return "", err
	}
	config := cloneImageBuildOptionsForBuildingStage(parb.b.options)
	newb, err := NewBuilder(parb.b.clientCtx, config, parb.b.docker, parb.b.context, nil)
	if err != nil {
		return "", err
	}
	newb.dockerfile = df
	imageID, err := newb.build(parb.stdout, parb.stderr, parb.output)
	if err != nil {
		return "", err
	}
	logrus.Debugf("[PARALLEL BUILDER] Built stage %d as %s", idx, imageID)
	fmt.Fprintf(parb.stdout, "Built stage %d as %s\n", idx, imageID)
	return imageID, nil
}
