package main

import (
	"fmt"
	"grapher/internal/graph"
)

func main() {
	// Create a new graph
	g := graph.New[any]()

	// Add nodes with any type of data
	g.AddNode("1", "String data")
	g.AddNode("2", 42)
	g.AddNode("3", map[string]any{
		"name":  "Complex Data",
		"value": 100,
	})
	g.AddNode("4", map[string]any{
		"name":  "Complex Data",
		"value": 100,
	})
	g.AddNode("5", map[string]any{
		"name":  "Complex Data",
		"value": 100,
	})

	// Add edges
	g.AddEdge("1", "2", 1.0)
	g.AddEdge("2", "3", 2.5)
	g.AddEdge("2", "4", 2.5)
	g.AddEdge("3", "4", 0.5)
	g.AddEdge("4", "1", 0.5)
	g.AddEdge("4", "5", 0.5)
	g.AddEdge("5", "1", 0.5)

	// Get node data
	node, _ := g.GetNode("1")
	fmt.Println("Node 1 data:", node.Data)

	// Get neighbors
	neighbors, _ := g.GetEdges("1")
	fmt.Println("Node 1 neighbors:", neighbors)

	// Save to file
	g.SaveToFile("my_graph.json")

	// Load from file
	newGraph := graph.New[any]()
	newGraph.LoadFromFile("my_graph.json")
	fmt.Println("data:", newGraph)
}
