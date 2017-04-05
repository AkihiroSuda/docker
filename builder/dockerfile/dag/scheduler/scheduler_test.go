package scheduler

import (
	"testing"

	"github.com/docker/docker/builder/dockerfile/dag"
	"github.com/docker/docker/pkg/testutil/assert"
)

func TestDetermineSchedule(t *testing.T) {
	g := &dag.Graph{
		Nodes: []dag.Node{0, 1, 2, 3, 4, 5},
		Edges: []dag.Edge{
			{Depender: 2, Dependee: 0},
			{Depender: 3, Dependee: 1},
			{Depender: 4, Dependee: 2},
			{Depender: 5, Dependee: 2},
		},
	}
	schedRoot := &ScheduleRoot{
		Children: []*Schedule{
			{
				Node: dag.Node(0),
				Children: []*Schedule{
					{
						Node: dag.Node(2),
						Children: []*Schedule{
							{Node: dag.Node(4)},
							{Node: dag.Node(5)},
						},
					},
				},
			},
			{
				Node: dag.Node(1),
				Children: []*Schedule{
					{
						Node: dag.Node(3),
					},
				},
			},
		},
	}
	assert.DeepEqual(t, DetermineSchedule(g), schedRoot)
}
