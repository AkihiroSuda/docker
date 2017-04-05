package scheduler

import (
	"math/rand"
	"testing"
	"time"

	"github.com/docker/docker/pkg/dag"
	"github.com/docker/docker/pkg/testutil/assert"
)

func TestExecuteSchedule(t *testing.T) {
	testExecuteSchedule(t, 0)
}

func testExecuteSchedule(t *testing.T, parallelism int) []dag.Node {
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
						Children: []*Schedule{
							// even though duplicated, node should be executed only once
							{Node: dag.Node(5)},
						},
					},
				},
			},
		},
	}
	c := make(chan dag.Node, 6)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	err := ExecuteSchedule(schedRoot, parallelism, func(n dag.Node) error {
		t.Logf("executing node %d", n)
		time.Sleep(time.Duration(rnd.Int63n(int64(100 * time.Millisecond))))
		c <- n
		return nil
	})
	assert.NilError(t, err)
	close(c)
	var got []dag.Node
	for n := range c {
		got = append(got, n)
	}
	assert.Equal(t, len(got), 6)
	assert.Equal(t, indexOf(got, 0) < indexOf(got, 2), true)
	assert.Equal(t, indexOf(got, 2) < indexOf(got, 4), true)
	assert.Equal(t, indexOf(got, 2) < indexOf(got, 5) || indexOf(got, 3) < indexOf(got, 5), true)
	assert.Equal(t, indexOf(got, 1) < indexOf(got, 3), true)
	return got
}

func indexOf(nodes []dag.Node, node dag.Node) int {
	for i, n := range nodes {
		if n == node {
			return i
		}
	}
	panic("node not found")
}
