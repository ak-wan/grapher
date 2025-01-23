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

	// 解析查询
	strQuery := "MATCH (A {data: 'Node A'})-[*]->(n) RETURN A, n;"
	q, err := cypher.ParseQuery(strQuery)
	if err != nil {
		fmt.Printf("Parse error: %s\n", err)
		return
	}

	// 执行查询
	results, err := cypher.ExecuteQuery(q, g)
	if err != nil {
		fmt.Printf("Execute error: %s\n", err)
		return
	}

	// 打印结果
	fmt.Println(results)
}
