// main.go
package main

import (
	"fmt"
	"grapher/internal/graph"
	"grapher/pkg/cypher"
)

func main() {
	// 加载图
	g := graph.New[string]()
	if err := g.LoadFromFile("my_graph.json"); err != nil {
		fmt.Println("Error loading graph:", err)
		return
	}

	// 打印节点和边数量
	fmt.Printf("Loaded %d nodes\n", len(g.AllNodes()))

	// 打印前3个节点的详细信息
	for i, node := range g.AllNodes() {
		fmt.Printf("Node %d: ID=%v, Data=%v, Labels=%v, Props=%v\n", i, node.ID, node.Data, node.Labels, node.Properties)
	}

	// 解析查询
	strQuery := "MATCH (A {data: 'Node A'})-[*]->(n) RETURN A, n;"
	q, err := cypher.ParseQuery(strQuery)
	if err != nil {
		fmt.Printf("Parse error: %s\n", err)
		return
	}
	// 打印解析后的查询结构
	fmt.Printf("Parsed Query:\n%#v\n", q.Root)

	// 执行查询
	results, err := cypher.ExecuteQuery(q, g)
	if err != nil {
		fmt.Printf("Execute error: %s\n", err)
		return
	}

	// 打印结果
	fmt.Println(results)
}
