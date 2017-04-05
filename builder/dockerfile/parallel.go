package dockerfile

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	// "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/builder"
	"github.com/docker/docker/builder/dockerfile/dag"
	"github.com/docker/docker/builder/dockerfile/parser"
	"golang.org/x/net/context"
)

// parallelBuilder is tricky
//  - Parses the Dockerfile, create stage DAG, and determine scheduling
//  - Calls NewBuilder() for each of stage in parallel, with Parallel=false, so as to ensure caches
//  - Calls NewBuilder() for the entire dockerfile, with Parallel=false
type parallelBuilder struct {
	stages []*stage
	daggy  *dag.Graph
}

type stage struct {
	startLine  int
	endLine    int
	dockerfile []byte
	name       string
	dependency []string
}

func cutLineRange(b []byte, startLine, endLine int) []byte {
	sep := "\n"
	split := strings.Split(string(b), sep)
	return []byte(strings.Join(split[startLine:endLine], sep))
}

func parseStageName(fromNode *parser.Node) string {
	image := fromNode.Next
	as := image.Next
	if as == nil {
		return ""
	}
	stageName := as.Next
	if stageName == nil {
		// likely to be a broken dockerfile, should we error out, FIXME
		return ""
	}
	return stageName.Value
}

func parseDependency(copyNode *parser.Node) string {
	for _, fl := range copyNode.Flags {
		if fl == "--from" {
			return copyNode.Next.Value
		}
	}
	return ""
}

func parseStages(dockerfile []byte) ([]*stage, error) {
	directive := parser.Directive{
		EscapeSeen:           false,
		LookingForDirectives: true,
	}
	parser.SetEscapeToken(parser.DefaultEscapeToken, &directive)
	rootNode, err := parser.Parse(ioutil.NopCloser(bytes.NewReader(dockerfile)), &directive)
	if err != nil {
		return nil, err
	}
	var stages []*stage
	var st *stage
	for i, n := range rootNode.Children {
		if i == len(rootNode.Children)-1 && st != nil {
			st.endLine = n.EndLine
			st.dockerfile = cutLineRange(dockerfile, st.startLine-1, st.endLine)
			stages = append(stages, st)
		}
		switch n.Value {
		case "from":
			if st != nil {
				st.endLine = n.StartLine - 1
				st.dockerfile = cutLineRange(dockerfile, st.startLine-1, st.endLine)
				stages = append(stages, st)
			}
			st = &stage{
				startLine: n.StartLine,
				name:      parseStageName(n),
			}
		case "copy":
			dependency := parseDependency(n)
			if dependency != "" {
				st.dependency = append(st.dependency, dependency)
			}
		}
	}

	return stages, nil
}

func createDAG(stages []*stage) (*dag.Graph, error) {
	g := &dag.Graph{
		Nodes: len(stages),
	}
	dagNodeByName := make(map[string]dag.Node, 0)
	for i, st := range stages {
		if st.name != "" {
			dagNodeByName[st.name] = dag.Node(i)
		}
	}
	for i, st := range stages {
		for _, dep := range st.dependency {
			depender := dag.Node(i)
			dependee, ok := dagNodeByName[dep]
			if !ok {
				// this is not an error,  typically when
				// COPY --from=registry.example.com/image ...
				continue
			}
			g.Edges = append(g.Edges, dag.Edge{
				Depender: depender,
				Dependee: dependee,
			})
		}
	}
	return g, nil
}

// newParallelBuilder instantiates parallelBuilder
func newParallelBuilder(clientCtx context.Context, config *types.ImageBuildOptions, backend builder.Backend, buildContext builder.Context, dockerfile io.ReadCloser) (b *parallelBuilder, err error) {
	return &parallelBuilder{}, nil
}

func (b *parallelBuilder) build(stdout io.Writer, stderr io.Writer, out io.Writer) (string, error) {
	return "", nil
}
