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

	// 解析查询
	strQuery := "MATCH (x {data: 'Node A'})-[*]->(y {data: 'Node C'}) RETURN y;"
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
		fmt.Printf("Execution error: %s\n", err)
		return
	}

	// 打印结果
	fmt.Println("\nQuery Results:")
	for i, res := range results {
		fmt.Printf("Result %d:\n", i+1)
		for k, v := range res {
			fmt.Printf("  %s = %v\n", k, v)
		}
	}

}
