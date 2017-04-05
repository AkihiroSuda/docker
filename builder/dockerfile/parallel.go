package dockerfile

import (
	"fmt"
	
//	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/builder/dockerfile/dag"
	"github.com/docker/docker/builder/dockerfile/parallel"
)

// parallelBuilder is tricky
//  - Parses the Dockerfile, create stage DAG, and determine scheduling
//  - Calls NewBuilder() for each of stage in parallel, with Parallel=false, so as to ensure caches
//  - Return the result of building the last stage
type parallelBuilder struct {
	// common
	b *Builder
	// specific to parallel builder
	stages []*parallel.Stage
	daggy  *dag.Graph
}


func cloneImageBuildOptionsForCachingStage(c *types.ImageBuildOptions) *types.ImageBuildOptions {
	return &types.ImageBuildOptions{
		BuildArgs: c.BuildArgs,
		// TODO: Context?
		Parallel: false,
	}
}


func (b *parallelBuilder) buildStages() ([]string, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
