// Package dag provides dag
package dag

// Node is a zero-indexed, consecutive integer that denotes a node
type Node int

// Edge is a edge
type Edge struct {
	Depender Node // from
	Dependee Node // to
}

// Graph is a node
type Graph struct {
	Nodes int
	Edges []Edge
}
