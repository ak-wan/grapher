package traverse

import (
	"errors"
	"grapher/internal/graph"
	"testing"
)

func TestDFS(t *testing.T) {
	t.Run("基础遍历", TestDFSBasic)
	t.Run("逆向遍历", TestDFSIncoming)
	t.Run("深度限制", TestDFSWithMaxDepth)
	t.Run("条件遍历", TestRangeTraversal)
	t.Run("错误处理", TestDFSErrorCases)
}

// 构建增强版测试图，包含属性
func buildEnhancedGraph() *graph.Graph[string] {
	g := graph.New[string]()

	nodes := map[string]map[string]string{
		"A": {"type": "start", "group": "1"},
		"B": {"type": "middle", "group": "1"},
		"C": {"type": "middle", "group": "2"},
		"D": {"type": "middle", "group": "2"},
		"E": {"type": "end", "group": "3"},
		"F": {"type": "end", "group": "3"},
	}

	for id, props := range nodes {
		g.AddNode(id, props)
	}

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

func TestDFSBasic(t *testing.T) {
	g := buildEnhancedGraph()
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
	expected := map[string]bool{"A": true, "B": true, "C": true, "D": true, "E": true, "F": true}
	for _, id := range result {
		delete(expected, id)
	}
	if len(expected) > 0 {
		t.Errorf("未访问所有节点，缺失: %v", expected)
	}

	// 验证DFS基本特性（深度优先）
	validOrders := []struct {
		path []string
	}{
		{[]string{"A", "D", "E", "F", "B", "C"}},
		{[]string{"A", "B", "C", "D", "E", "F"}},
	}
	match := false
	for _, vo := range validOrders {
		if isPathEqual(result, vo.path) {
			match = true
			break
		}
	}
	if !match {
		t.Errorf("无效的DFS顺序: %v", result)
	}
}

func TestDFSIncoming(t *testing.T) {
	g := buildEnhancedGraph()
	iter, err := NewDFS(g, "F", WithDirection[string](Incoming))
	if err != nil {
		t.Fatalf("创建迭代器失败: %v", err)
	}

	var result []string
	iter.Iterate(func(n *graph.Node[string]) error {
		result = append(result, n.ID)
		return nil
	})

	// 验证逆向路径
	validPaths := [][]string{
		{"F", "E", "D", "C", "B", "A"},
		{"F", "C", "B", "A", "E", "D"},
	}
	valid := false
	for _, path := range validPaths {
		if isPathEqual(result, path) {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("无效的逆向路径: %v", result)
	}
}

func TestDFSWithMaxDepth(t *testing.T) {
	g := buildEnhancedGraph()
	iter, err := NewDFS(g, "A", WithMaxDepth[string](2))
	if err != nil {
		t.Fatalf("创建迭代器失败: %v", err)
	}

	var result []string
	iter.Iterate(func(n *graph.Node[string]) error {
		result = append(result, n.ID)
		return nil
	})

	// 验证深度限制
	allowed := map[string]bool{"A": true, "B": true, "D": true, "C": true, "E": true}
	for _, id := range result {
		if !allowed[id] {
			t.Errorf("出现超出深度的节点: %s", id)
		}
		if id == "F" {
			t.Error("不应包含深度3的节点F")
		}
	}
}

// 新增辅助函数验证子路径
func isSubPath(got []string, validPaths [][]string) bool {
NEXT_PATH:
	for _, path := range validPaths {
		if len(got) > len(path) {
			continue
		}
		for i := range got {
			if got[i] != path[i] {
				continue NEXT_PATH
			}
		}
		return true
	}
	return false
}

func TestDFSErrorCases(t *testing.T) {
	g := buildEnhancedGraph()

	t.Run("无效起点", func(t *testing.T) {
		_, err := NewDFS(g, "X")
		if !errors.Is(err, graph.ErrNodeNotFound) {
			t.Errorf("预期错误 %v, 实际 %v", graph.ErrNodeNotFound, err)
		}
	})

	t.Run("遍历中断", func(t *testing.T) {
		iter, _ := NewDFS(g, "A")
		err := iter.Iterate(func(n *graph.Node[string]) error {
			return errors.New("模拟错误")
		})
		if err == nil {
			t.Error("预期错误未返回")
		}
	})
}

// 辅助函数
func isPathEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isUnorderedEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int)
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		if counts[s] == 0 {
			return false
		}
		counts[s]--
	}
	return true
}

// 新增缺失的验证函数
func isValidDFSPath(got []string, validPaths [][]string) bool {
	for _, path := range validPaths {
		if isPathEqual(got, path) {
			return true
		}
	}
	return false
}

// 更新后的测试用例
func TestRangeTraversal(t *testing.T) {
	g := buildEnhancedGraph()

	t.Run("简单范围遍历-A到C", func(t *testing.T) {
		iter, _ := NewDFS(g, "A",
			WithRangeFilter(
				func(n *graph.Node[string]) bool { return n.ID == "A" },
				func(n *graph.Node[string]) bool { return n.ID == "C" },
			),
			WithDirection[string](Outgoing),
		)

		var result []string
		iter.Iterate(func(n *graph.Node[string]) error {
			result = append(result, n.ID)
			return nil
		})

		// 定义有效路径集合
		validPaths := [][]string{
			{"A", "B", "C"}, // 唯一有效路径
		}

		// 验证结果
		if !isValidDFSPath(result, validPaths) {
			t.Errorf("无效的DFS路径: %v", result)
		}
	})

	t.Run("类型范围遍历-start到end", func(t *testing.T) {
		iter, _ := NewDFS(g, "A",
			WithRangeFilter(
				func(n *graph.Node[string]) bool { return n.Properties["type"] == "start" },
				func(n *graph.Node[string]) bool { return n.Properties["type"] == "end" },
			),
			WithDirection[string](Outgoing),
		)

		var result []string
		iter.Iterate(func(n *graph.Node[string]) error {
			result = append(result, n.ID)
			return nil
		})

		// 定义有效路径模式
		validPatterns := []struct {
			required []string
			optional []string
		}{
			{
				required: []string{"A", "B", "C", "F"},
				optional: []string{"D", "E"},
			},
			{
				required: []string{"A", "D", "E", "F"},
				optional: []string{"B", "C"},
			},
		}

		// 验证必须包含的节点
		requiredCheck := make(map[string]bool)
		for _, p := range validPatterns {
			for _, n := range p.required {
				requiredCheck[n] = true
			}
		}
		for _, n := range result {
			delete(requiredCheck, n)
		}
		if len(requiredCheck) > 0 {
			t.Errorf("缺失必要节点: %v", requiredCheck)
		}

		// 验证路径有效性
		valid := false
		for _, pattern := range validPatterns {
			if hasSubsequence(result, pattern.required) {
				valid = true
				break
			}
		}
		if !valid {
			t.Errorf("无效的路径模式: %v", result)
		}
	})
}

// 子序列验证（原有函数保持不变）
func hasSubsequence(got []string, sub []string) bool {
	if len(sub) == 0 {
		return true
	}

	i := 0
	for _, v := range got {
		if v == sub[i] {
			i++
			if i == len(sub) {
				return true
			}
		}
	}
	return false
}
