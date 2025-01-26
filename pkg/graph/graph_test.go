// graph_test.go
package graph

import (
	"errors"
	"math/rand"
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

// 基准测试组
func BenchmarkGraph(b *testing.B) {
	b.Run("添加节点", benchmarkAddNode)
	b.Run("添加边", benchmarkAddEdge)
	b.Run("随机任务", benchmarkMixedWorkload)
}

// 节点操作测试（已适配新结构）
func testNodeOperations(t *testing.T) {
	t.Parallel()

	g := New[string]()

	t.Run("AddNode", func(t *testing.T) {
		// 正常添加（使用properties参数）
		if err := g.AddNode("A", map[string]string{"name": "NodeA"}); err != nil {
			t.Errorf("AddNode failed: %v", err)
		}

		// 重复添加
		err := g.AddNode("A", map[string]string{"name": "Another"})
		if !errors.Is(err, ErrNodeExists) {
			t.Errorf("Expected ErrNodeExists, got %v", err)
		}

		// 空ID
		if err := g.AddNode("", map[string]string{}); !errors.Is(err, ErrInvalidInput) {
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
		g.AddNode("A", map[string]string{"name": "NodeA"})
		g.AddNode("B", map[string]string{"name": "NodeB"})
		g.AddEdge("A", "B", 1.0)
		if err := g.RemoveNode("A"); err != nil {
			t.Fatal(err)
		}
		if edges, _ := g.GetOutEdges("A"); len(edges) > 0 {
			t.Error("Related edges not cleaned up")
		}
	})
}

// 边操作测试（已适配新结构）
func testEdgeOperations(t *testing.T) {
	t.Parallel()

	g := New[string]()
	g.AddNode("A", map[string]string{})
	g.AddNode("B", map[string]string{})

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

// 并发安全测试（已适配新结构）
func testConcurrency(t *testing.T) {
	t.Parallel()

	g := New[int]()
	const numNodes = 1000
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numNodes; i++ {
			_ = g.AddNode(string(rune(i)), map[string]int{"value": i})
		}
	}()

	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numNodes; i++ {
			_ = g.RemoveNode(string(rune(i)))
		}
	}()

	wg.Wait()

	if len(g.nodes) != 0 {
		t.Errorf("Expected empty graph, got %d nodes", len(g.nodes))
	}
}

// 混合读写操作测试（已适配新结构）
func testMixedConcurrency(t *testing.T) {
	g := New[int]()
	var wg sync.WaitGroup
	const numWorkers = 100

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = g.AddNode(string(rune(id)), map[string]int{"id": id})
			_, _ = g.GetNode(string(rune(id)))
			_ = g.RemoveNode(string(rune(id)))
		}(i)
	}

	wg.Wait()

	if len(g.nodes) > 0 {
		t.Errorf("Expected empty graph, got %d nodes", len(g.nodes))
	}
}

// 持久化测试（已适配新结构）
func testPersistence(t *testing.T) {
	t.Parallel()

	const testFile = "test_graph.json"
	defer os.Remove(testFile)

	orig := New[float64]()
	orig.AddNode("A", map[string]float64{"value": 1.1})
	orig.AddNode("B", map[string]float64{"value": 2.2})
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

		nodeA, _ := loaded.GetNode("A")
		if val, ok := nodeA.Properties["value"]; !ok || val != 1.1 {
			t.Errorf("Expected 1.1, got %v", val)
		}

		edges, _ := loaded.GetOutEdges("A")
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

// 基准测试：单线程添加节点
func benchmarkAddNode(b *testing.B) {
	g := New[string]()
	properties := map[string]string{"type": "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune(i % 1000)) // 限制ID范围模拟更新操作
		_ = g.AddNode(id, properties)
	}
}

// 基准测试：单线程添加边
func benchmarkAddEdge(b *testing.B) {
	g := New[int]()
	nodes := 1000
	for i := 0; i < nodes; i++ {
		g.AddNode(string(rune(i)), map[string]int{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		from := string(rune(rand.Intn(nodes)))
		to := string(rune(rand.Intn(nodes)))
		_ = g.AddEdge(from, to, 1)
	}
}

// 基准测试：混合工作负载
func benchmarkMixedWorkload(b *testing.B) {
	g := New[string]()
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		opCounter := 0
		for pb.Next() {
			opCounter++
			id := string(rune(rand.Intn(1000)))

			// 混合操作类型
			switch opCounter % 4 {
			case 0:
				mu.Lock()
				_ = g.AddNode(id, map[string]string{"id": id})
				mu.Unlock()
			case 1:
				mu.Lock()
				_, _ = g.GetNode(id)
				mu.Unlock()
			case 2:
				mu.Lock()
				_ = g.RemoveNode(id)
				mu.Unlock()
			case 3:
				from := string(rune(rand.Intn(1000)))
				to := string(rune(rand.Intn(1000)))
				mu.Lock()
				_ = g.AddEdge(from, to, 1.0)
				mu.Unlock()
			}
		}
	})
}
