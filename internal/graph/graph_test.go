// graph_test.go
package graph

import (
	"errors"
	"os"
	"sync"
	"testing"
)

func TestGraph(t *testing.T) {
	t.Run("测试操作节点", testNodeOperations)
	t.Run("测试边操作", testEdgeOperations)
	t.Run("并发", testConcurrency)
	t.Run("持久化", testPersistence)
	t.Run("混合读写", testMixedConcurrency)
}

// 节点操作测试
func testNodeOperations(t *testing.T) {
	t.Parallel()

	g := New[string]()

	t.Run("AddNode", func(t *testing.T) {
		// 正常添加
		if err := g.AddNode("A", "NodeA"); err != nil {
			t.Errorf("AddNode failed: %v", err)
		}

		// 重复添加
		err := g.AddNode("A", "NodeA")
		if !errors.Is(err, ErrNodeExists) {
			t.Errorf("Expected ErrNodeExists, got %v", err)
		}

		// 空ID
		if err := g.AddNode("", "Empty"); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("RemoveNode", func(t *testing.T) {
		// 正常删除
		if err := g.RemoveNode("A"); err != nil {
			t.Errorf("RemoveNode failed: %v", err)
		}

		// 删除不存在节点
		err := g.RemoveNode("B")
		if !errors.Is(err, ErrNodeNotFound) {
			t.Errorf("Expected ErrNodeNotFound, got %v", err)
		}

		// 验证关联边删除
		g.AddNode("A", "NodeA")
		g.AddNode("B", "NodeB")
		g.AddEdge("A", "B", 1.0)
		if err := g.RemoveNode("A"); err != nil {
			t.Fatal(err)
		}
		if edges, _ := g.GetEdges("B"); len(edges) > 0 {
			t.Error("Related edges not cleaned up")
		}
	})
}

// 边操作测试
func testEdgeOperations(t *testing.T) {
	t.Parallel()

	g := New[string]()
	g.AddNode("A", "")
	g.AddNode("B", "")

	t.Run("AddEdge", func(t *testing.T) {
		// 正常添加
		if err := g.AddEdge("A", "B", 1.5); err != nil {
			t.Error(err)
		}

		// 重复添加
		err := g.AddEdge("A", "B", 2.0)
		if !errors.Is(err, ErrEdgeExists) {
			t.Errorf("Expected ErrEdgeExists, got %v", err)
		}

		// 无效节点
		cases := []struct{ from, to string }{
			{"X", "B"},
			{"A", "Y"},
		}
		for _, c := range cases {
			err := g.AddEdge(c.from, c.to, 1.0)
			if !errors.Is(err, ErrNodeNotFound) {
				t.Errorf("Expected ErrNodeNotFound, got %v", err)
			}
		}
	})

	t.Run("UpdateEdge", func(t *testing.T) {
		// 正常更新
		if err := g.UpdateEdge("A", "B", 2.0); err != nil {
			t.Error(err)
		}

		// 更新不存在边
		err := g.UpdateEdge("B", "A", 1.0)
		if !errors.Is(err, ErrEdgeNotFound) {
			t.Errorf("Expected ErrEdgeNotFound, got %v", err)
		}
	})

	t.Run("RemoveEdge", func(t *testing.T) {
		// 正常删除
		if err := g.RemoveEdge("A", "B"); err != nil {
			t.Error(err)
		}

		// 删除不存在边
		err := g.RemoveEdge("A", "B")
		if !errors.Is(err, ErrEdgeNotFound) {
			t.Errorf("Expected ErrEdgeNotFound, got %v", err)
		}
	})
}

// 并发安全测试
func testConcurrency(t *testing.T) {
	t.Parallel()

	g := New[int]()
	const numNodes = 1000
	var wg sync.WaitGroup

	// 并发添加节点
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numNodes; i++ {
			_ = g.AddNode(string(rune(i)), i)
		}
	}()

	wg.Wait() // 等待添加完成

	// 并发删除节点
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numNodes; i++ {
			_ = g.RemoveNode(string(rune(i)))
		}
	}()

	wg.Wait()

	// 验证最终状态
	if len(g.nodes) != 0 {
		t.Errorf("Expected empty graph, got %d nodes", len(g.nodes))
	}
}

// 混合读写操作测试
func testMixedConcurrency(t *testing.T) {
	g := New[int]()
	var wg sync.WaitGroup
	const numWorkers = 100

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 混合操作
			_ = g.AddNode(string(rune(id)), id)
			_, _ = g.GetNode(string(rune(id)))
			_ = g.RemoveNode(string(rune(id)))
		}(i)
	}

	wg.Wait()

	// 验证最终无节点残留
	if len(g.nodes) > 0 {
		t.Errorf("Expected empty graph, got %d nodes", len(g.nodes))
	}
}

// 持久化测试
func testPersistence(t *testing.T) {
	t.Parallel()

	const testFile = "test_graph.json"
	defer os.Remove(testFile)

	orig := New[float64]()
	orig.AddNode("A", 1.1)
	orig.AddNode("B", 2.2)
	orig.AddEdge("A", "B", 3.14)

	t.Run("Save", func(t *testing.T) {
		if err := orig.SaveToFile(testFile); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Load", func(t *testing.T) {
		loaded := New[float64]()
		if err := loaded.LoadFromFile(testFile); err != nil {
			t.Fatal(err)
		}

		// 验证节点
		nodeA, _ := loaded.GetNode("A")
		if nodeA.Data != 1.1 {
			t.Errorf("Expected 1.1, got %v", nodeA.Data)
		}

		// 验证边
		edges, _ := loaded.GetEdges("A")
		if len(edges) != 1 || edges[0].Weight != 3.14 {
			t.Error("Edge data mismatch")
		}
	})

	t.Run("InvalidFile", func(t *testing.T) {
		err := orig.LoadFromFile("non_existent.json")
		if err == nil {
			t.Error("Expected error for missing file")
		}
	})
}
