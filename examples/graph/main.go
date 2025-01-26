package main

import (
	"fmt"
	"log"

	"grapher/pkg/graph"
)

func main() {
	// 创建新图实例
	g := graph.New[any]()

	// 添加带属性的节点
	addNodes(g)

	// 添加边
	addEdges(g)

	// 获取并打印节点属性
	printNodeProperties(g)

	// 获取并打印出边
	printOutEdges(g)

	fmt.Printf("%v\n", g.AllNodes())
	g.SaveToFile("my_graph.json")
}

func addNodes(g *graph.Graph[any]) {
	nodes := []struct {
		id    string
		props map[string]any
	}{
		{"1", map[string]any{"data": "String data"}},
		{"2", map[string]any{"value": 42}},
		{"3", map[string]any{
			"name":  "Complex Data",
			"value": 100,
		}},
		{"4", map[string]any{
			"name":  "Complex Data",
			"value": 100,
		}},
		{"5", map[string]any{
			"name":  "Complex Data",
			"value": 100,
		}},
	}

	for _, n := range nodes {
		if err := g.AddNode(n.id, n.props); err != nil {
			log.Fatalf("添加节点失败: %v", err)
		}
	}
}

func addEdges(g *graph.Graph[any]) {
	edges := []struct {
		from   string
		to     string
		weight float64
	}{
		{"1", "2", 1.0},
		{"2", "3", 2.5},
		{"2", "4", 2.5},
		{"3", "4", 0.5},
		{"4", "1", 0.5},
		{"4", "5", 0.5},
		{"5", "1", 0.5},
	}

	for _, e := range edges {
		if err := g.AddEdge(e.from, e.to, e.weight); err != nil {
			log.Fatalf("添加边失败: %v", err)
		}
	}
}

func printNodeProperties(g *graph.Graph[any]) {
	node, err := g.GetNode("1")
	if err != nil {
		log.Fatalf("获取节点失败: %v", err)
	}
	fmt.Printf("节点 1 属性: %+v\n", node.Properties)
}

func printOutEdges(g *graph.Graph[any]) {
	edges, err := g.GetOutEdges("1")
	if err != nil {
		log.Fatalf("获取出边失败: %v", err)
	}

	fmt.Println("\n节点 1 的出边:")
	for _, edge := range edges {
		fmt.Printf("-> %s (权重: %.1f)\n", edge.To, edge.Weight)
	}
}
