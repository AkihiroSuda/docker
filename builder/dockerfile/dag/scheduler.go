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
	return nil
}
