package dockerfile

import (
	"testing"

	"github.com/docker/docker/builder/dockerfile/dag"
	"github.com/docker/docker/pkg/testutil/assert"
)

var parallelTestDockerfile = []byte(`FROM busybox AS foo
RUN dummy-foo \
apple \
pineapple
FROM busybox AS bar
RUN dummy-bar
FROM busybox
COPY --from foo /dummy-foo-out \
/x
FROM busybox
COPY --from bar /dummy-bar-out /x
COPY --from docker.io/library/blah /blah /y`)

// TODO: make sure it does not panic for dockerfiles under testfiles dir
func TestParallelParseStages(t *testing.T) {
	stages, err := parseStages(parallelTestDockerfile)
	assert.NilError(t, err)
	t.Logf("parsed %d stages", len(stages))
	assert.Equal(t, len(stages), 4)
	for i, st := range stages {
		t.Logf("=== Stage %d ===", i)
		t.Logf("start line(1-indexed): %d", st.startLine)
		t.Logf("end line: %d", st.endLine)
		t.Logf("file: %q", st.dockerfile)
		t.Logf("name: %q", st.name)
		t.Logf("dependency: %v", st.dependency)
	}
	assert.Equal(t, stages[0].startLine, 1)
	assert.Equal(t, stages[0].endLine, 4)
	assert.Equal(t, stages[0].name, "foo")
	assert.Equal(t, len(stages[0].dependency), 0)
	assert.Equal(t, stages[1].startLine, 5)
	assert.Equal(t, stages[1].endLine, 6)
	assert.Equal(t, stages[1].name, "bar")
	assert.Equal(t, len(stages[1].dependency), 0)
	assert.Equal(t, stages[2].startLine, 7)
	assert.Equal(t, stages[2].endLine, 9)
	assert.Equal(t, stages[2].name, "")
	assert.DeepEqual(t, stages[2].dependency, []string{"foo"})
	assert.Equal(t, stages[3].startLine, 10)
	assert.Equal(t, stages[3].endLine, 12)
	assert.Equal(t, stages[3].name, "")
	assert.DeepEqual(t, stages[3].dependency, []string{"bar", "docker.io/library/blah"})

	dagGraph, err := createDAG(stages)
	assert.NilError(t, err)
	t.Logf("graph: %#v", dagGraph)
	assert.DeepEqual(t, dagGraph,
		&dag.Graph{
			Nodes: 4,
			Edges: []dag.Edge{
				{Depender: dag.Node(2), Dependee: dag.Node(0)},
				{Depender: dag.Node(3), Dependee: dag.Node(1)},
			},
		})
}
