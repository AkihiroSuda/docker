package dag

import (
	"testing"

	"github.com/docker/docker/pkg/testutil/assert"
)

func TestDetermineSchedule(t *testing.T) {
	g := &Graph{
		Nodes: []Node{0, 1, 2, 3, 4, 5},
		Edges: []Edge{
			{Depender: 2, Dependee: 0},
			{Depender: 3, Dependee: 1},
			{Depender: 4, Dependee: 2},
			{Depender: 5, Dependee: 2},
		},
	}
	schedRoot := &ScheduleRoot{
		Children: []*Schedule{
			{
				Node: Node(0),
				Children: []*Schedule{
					{
						Node: Node(2),
						Children: []*Schedule{
							{Node: Node(4)},
							{Node: Node(5)},
						},
					},
				},
			},
			{
				Node: Node(1),
				Children: []*Schedule{
					{
						Node: Node(3),
					},
				},
			},
		},
	}
	assert.DeepEqual(t, DetermineSchedule(g), schedRoot)
}
