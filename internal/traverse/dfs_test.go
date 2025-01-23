// traverse/dfs_test.go
package traverse

import (
	"grapher/internal/graph"
	"testing"
)

func TestDFS(t *testing.T) {
	t.Run("测试完整向下遍历所有节点", TestDFSOutgoingFull)
	t.Run("测试逆向遍历链路", TestDFSIncomingPath)
	t.Run("验证最大深度控制逻辑", TestDFSWithMaxDepth)
	t.Run("测试无效节点等异常场景", TestDFSErrorCases)
}

// 构建测试用图结构
// A -> D -> E
// |    ^    |
// v    |    v
// B -> C -> F
func buildTestGraph() *graph.Graph[string] {
	g := graph.New[string]()

	// 添加节点
	for _, id := range []string{"A", "B", "C", "D", "E", "F"} {
		g.AddNode(id, id)
	}

	// 添加边
	edges := []struct{ from, to string }{
		{"A", "B"}, {"A", "D"},
		{"B", "C"},
		{"C", "D"}, {"C", "F"},
		{"D", "E"},
		{"E", "F"},
	}
	for _, e := range edges {
		g.AddEdge(e.from, e.to, 0)
	}
	return g
}

// 验证切片是否包含所有元素
func containsAll(got, want []string) bool {
	m := make(map[string]bool, len(got))
	for _, v := range got {
		m[v] = true
	}
	for _, v := range want {
		if !m[v] {
			return false
		}
	}
	return true
}

// 验证切片顺序是否匹配任一有效路径
func isValidDFSPath(got []string, validPaths [][]string) bool {
	for _, path := range validPaths {
		if len(got) != len(path) {
			continue
		}
		match := true
		for i := range got {
			if got[i] != path[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func TestDFSOutgoingFull(t *testing.T) {
	g := buildTestGraph()
	iter, err := NewDFS(g, "A")

	if err != nil {
		t.Fatalf("创建迭代器失败: %v", err)
	}

	var result []string
	iter.Iterate(func(n *graph.Node[string]) error {
		result = append(result, n.ID)
		return nil
	})

	// 验证访问所有节点
	if !containsAll(result, []string{"A", "B", "C", "D", "E", "F"}) {
		t.Errorf("未访问所有节点, 结果: %v", result)
	}

	// 验证可能的DFS顺序
	validPaths := [][]string{
		{"A", "D", "E", "F", "B", "C"},
	}
	
	if !isValidDFSPath(result, validPaths) {
		t.Errorf("无效的DFS顺序: %v", result)
	}
}

func TestDFSIncomingPath(t *testing.T) {
	g := buildTestGraph()
	iter, err := NewDFS[string](g, "F", WithDirection[string](Incoming), WithMaxDepth[string](2))
	if err != nil {
		t.Fatalf("创建迭代器失败: %v", err)
	}

	var result []string
	iter.Iterate(func(n *graph.Node[string]) error {
		result = append(result, n.ID)
		return nil
	})

	expected := []string{"F", "C", "E", "B", "D"}
	if !containsAll(result, expected) {
		t.Errorf("缺少预期节点, 结果: %v", result)
	}

	validPaths := [][]string{
		{"F", "E", "D", "C", "B"},
		{"F", "C", "B", "E", "D"},

	}
	if !isValidDFSPath(result, validPaths) {
		t.Errorf("无效的逆向路径: %v", result)
	}
}

func TestDFSWithMaxDepth(t *testing.T) {
	g := buildTestGraph()
	iter, err := NewDFS[string](g, "A", WithMaxDepth[string](2)) // 允许深度 0~2
	if err != nil {
		t.Fatalf("创建迭代器失败: %v", err)
	}

	var result []string
	iter.Iterate(func(n *graph.Node[string]) error {
		result = append(result, n.ID)
		return nil
	})

	// 正确的结果应包含 A(0), B(1), D(1), C(2), E(2)
	expectedNodes := []string{"A", "B", "D", "C", "E"}
	if !containsAll(result, expectedNodes) {
		t.Errorf("节点缺失，期望 %v，实际 %v", expectedNodes, result)
	}

	// 验证不包含深度超过 2 的节点（如 F）
	for _, id := range result {
		if id == "F" {
			t.Errorf("包含超出深度的节点: F")
		}
	}
}

func TestDFSErrorCases(t *testing.T) {
	// 测试无效节点
	t.Run("InvalidStartNode", func(t *testing.T) {
		g := buildTestGraph()
		_, err := NewDFS(g, "X")
		if err == nil {
			t.Error("预期错误未发生")
		}
	})

	// 测试空图
	t.Run("EmptyGraph", func(t *testing.T) {
		g := graph.New[string]()
		_, err := NewDFS(g, "A")
		if err == nil {
			t.Error("空图应返回错误")
		}
	})
}
