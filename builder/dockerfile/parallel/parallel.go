package parallel

import (
	"fmt"
	"strings"

	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/docker/docker/pkg/dag"
)

type Stage struct {
	// this Dockerfile does NOT contain dependency stages (so cannot be directly built)
	Dockerfile *parser.Node
	Name       string
	Dependency []string
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
			// Did we support this ever?
			return copyNode.Next.Value
		} else if strings.HasPrefix(fl, "--from=") {
			return strings.TrimPrefix(fl, "--from=")
		}
	}
	return ""
}

func ParseStages(rootNode *parser.Node) ([]*Stage, error) {
	var (
		stages []*Stage
		st     *Stage
	)
	for i, n := range rootNode.Children {
		if i == len(rootNode.Children)-1 && st != nil {
			stages = append(stages, st)
		}
		switch n.Value {
		case "from":
			if st != nil {
				stages = append(stages, st)
			}
			st = &Stage{
				Dockerfile: &parser.Node{
					Children: []*parser.Node{n},
				},
				Name: parseStageName(n),
			}
		case "copy":
			dependency := parseDependency(n)
			if dependency != "" {
				st.Dependency = append(st.Dependency, dependency)
			}
			fallthrough
		default:
			st.Dockerfile.Children = append(st.Dockerfile.Children, n)
		}
	}

	return stages, nil
}

func CreateDAG(stages []*Stage) (*dag.Graph, error) {
	g := &dag.Graph{}
	for i := range stages {
		g.AddNode(dag.Node(i))
	}
	dagNodeByName := make(map[string]dag.Node, 0)
	for i, st := range stages {
		if st.Name != "" {
			dagNodeByName[st.Name] = dag.Node(i)
		}
	}
	for i, st := range stages {
		for _, dep := range st.Dependency {
			depender := dag.Node(i)
			dependee, ok := dagNodeByName[dep]
			if !ok {
				// this is not an error,  typically when
				// COPY --from=registry.example.com/image ...
				continue
			}
			g.AddEdge(dag.Edge{
				Depender: depender,
				Dependee: dependee,
			})
		}
	}
	return g, nil
}

func lookupStage(stages []*Stage, name string) (int, *Stage) {
	for i, st := range stages {
		if st.Name == name {
			return i, st
		}
	}
	return -1, nil
}

// CreateDockerfileThatContainsDependencyStages really needs to be renamed :-(
func CreateDockerfileThatContainsDependencyStages(stages []*Stage, idx int) (*parser.Node, error) {
	if idx < 0 || idx > len(stages)-1 {
		return nil, fmt.Errorf("unknown stage index %d", idx)
	}
	st := stages[idx]
	df := &parser.Node{
		Children: st.Dockerfile.Children,
	}
	for _, depName := range st.Dependency {
		depIdx, _ := lookupStage(stages, depName)
		if depIdx < 0 {
			// this is not an error,  typically when
			// COPY --from=registry.example.com/image ...
			continue
		}
		dep, err := CreateDockerfileThatContainsDependencyStages(stages, depIdx)
		if dep == nil {
			return nil, err
		}
		df.Children = append(dep.Children, df.Children...)
	}
	return df, nil
}
