package dag

// Schedule denotes a schedule
type Schedule struct {
	Node Node
	// Children are executed in parallel after executing Node.
	// Sequential schedule can be expressed as a chain of single-children schedules.
	Children []*Schedule
}

type ScheduleRoot struct {
	Children []*Schedule
}

// DetermineSchedule is written without reading any paper and hence likely to be wrong :-(
func DetermineSchedule(g *Graph) *ScheduleRoot {
	schedRoot := &ScheduleRoot{
	}
	for _, compoRoot := range ComponentRoots(g) {
		subg := Subgraph(g, compoRoot)
		if subg != nil {
			schedRoot.Children = append(schedRoot.Children, determineSchedule(subg, compoRoot))
		}
	}
	return schedRoot
}

func determineSchedule(subg *Graph, subgRoot Node) *Schedule {
	s := &Schedule{
		Node: subgRoot,
	}
	for _, depender := range Dependers(subg, subgRoot) {
		subsubg := Subgraph(subg, depender)
		child := determineSchedule(subsubg, depender)
		if child != nil {
			s.Children = append(s.Children, child)
		}
	}
	return s
}
