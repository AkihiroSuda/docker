package parallel

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/builder/dockerfile/dag"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/docker/docker/pkg/testutil/assert"
)

// DAG should have 4 nodes, 2 edges (2->0, 3->1)
var testDockerfile1 = []byte(`FROM busybox AS foo
RUN echo dummy-foo \
apple \
pineapple > /dummy-foo-out
FROM busybox AS bar
RUN echo dummy-bar > /dummy-bar-out
FROM busybox
COPY --from foo /dummy-foo-out \
/x
RUN cat /x
FROM busybox
COPY --from bar /dummy-bar-out /x
COPY --from docker.io/library/nginx /etc/passwd /y
RUN cat /x /y`)

func parseDockerfile(t *testing.T, b []byte) *parser.Node {
	directive := parser.Directive{
		EscapeSeen:           false,
		LookingForDirectives: true,
	}
	parser.SetEscapeToken(parser.DefaultEscapeToken, &directive)
	rootNode, err := parser.Parse(ioutil.NopCloser(bytes.NewReader(b)), &directive)
	if err != nil {
		t.Fatal(err)
	}
	return rootNode
}

// TODO: make sure it does not panic for dockerfiles under testfiles dir
func TestParseStages(t *testing.T) {
	df := parseDockerfile(t, testDockerfile1)
	t.Logf("=== Input ===")
	t.Logf("dockerfile dump: %q", df.Dump())
	stages, err := ParseStages(df)
	assert.NilError(t, err)
	t.Logf("parsed %d stages", len(stages))
	assert.Equal(t, len(stages), 4)
	for i, st := range stages {
		t.Logf("")
		t.Logf("=== Stage %d ===", i)
		t.Logf("name: %q", st.Name)
		t.Logf("dependency: %v", st.Dependency)
		t.Logf("dockerfile dump: %q", st.Dockerfile.Dump())
		dfWithDep, err := CreateDockerfileThatContainsDependencyStages(stages, i)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("dockerfile dump with dep: %q", dfWithDep.Dump())
	}
	assert.Equal(t, stages[0].Name, "foo")
	assert.Equal(t, len(stages[0].Dependency), 0)
	assert.Equal(t, stages[1].Name, "bar")
	assert.Equal(t, len(stages[1].Dependency), 0)
	assert.Equal(t, stages[2].Name, "")
	assert.DeepEqual(t, stages[2].Dependency, []string{"foo"})
	assert.Equal(t, stages[3].Name, "")
	assert.DeepEqual(t, stages[3].Dependency, []string{"bar", "docker.io/library/nginx"})

	graph, err := CreateDAG(stages)
	assert.NilError(t, err)
	t.Logf("graph: %#v", graph)
	assert.DeepEqual(t, graph,
		&dag.Graph{
			Nodes: []dag.Node{0, 1, 2, 3},
			Edges: []dag.Edge{
				{Depender: 2, Dependee: 0},
				{Depender: 3, Dependee: 1},
			},
		})
}
