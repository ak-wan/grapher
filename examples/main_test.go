package main

import (
	"grapher/internal/cypher"
	"grapher/pkg/graph"
	"testing"
)

func TestMain(t *testing.T) {
	t.Run("数据加载", TestLoadGraph)
	t.Run("语句解析", TestParseQuery)
	t.Run("执行查询", TestExecuteQuery)
}

func TestLoadGraph(t *testing.T) {
	g := graph.New[string]()
	if err := g.LoadFromFile("data/cypher.json"); err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	// 验证节点数量
	if len(g.AllNodes()) != 6 {
		t.Errorf("预期6个节点，实际得到 %d", len(g.AllNodes()))
	}
}

func TestParseQuery(t *testing.T) {
	strQuery := "MATCH (x {data: 'Node A'})-[*]->(y {data: 'Node F'}) RETURN y;"
	q, err := cypher.ParseQuery(strQuery)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 验证模式匹配结构
	if len(q.Root.Reading) == 0 {
		t.Fatal("未解析到MATCH子句")
	}

	// 验证返回字段
	if len(q.Root.ReturnItems) != 1 || q.Root.ReturnItems[0].String() != "y" {
		t.Errorf("RETURN字段不匹配: %v", q.Root.ReturnItems)
	}
}

func TestExecuteQuery(t *testing.T) {
	// 初始化图
	g := graph.New[string]()
	if err := g.LoadFromFile("data/cypher.json"); err != nil {
		t.Fatal(err)
	}

	// 构造测试用例表
	tests := []struct {
		name     string
		query    string
		expected int    // 预期结果数量
		target   string // 预期节点ID
	}{
		{
			name:     "路径查找",
			query:    "MATCH (x {data: 'Node A'})-[*]->(y {data: 'Node F'}) RETURN y;",
			expected: 6,
			target:   "F",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, _ := cypher.ParseQuery(tt.query)
			results, err := cypher.ExecuteQuery(q, g)
			if err != nil {
				t.Fatal(err)
			}

			if len(results) != tt.expected {
				t.Fatalf("预期 %d 个结果，实际得到 %d", tt.expected, len(results))
			}

			// 验证结果内容
			if v, ok := results[5]["ID"]; ok {
				if v.(string) != tt.target {
					t.Errorf("预期节点ID %s，实际得到 %s", tt.target, v.(string))
				}
			} else {
				t.Error("用例校验失败")
			}
		})
	}
}
