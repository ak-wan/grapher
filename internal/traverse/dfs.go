package traverse

import (
	"fmt"
	"grapher/internal/graph"
)

// 完善后的DFS实现
// 添加类型定义
type DFSOption[T comparable] func(*DFS[T])

type stackItem[T any] struct {
	node  *graph.Node[T]
	depth int
}

type DFS[T comparable] struct {
	graph     *graph.Graph[T]
	stack     []stackItem[T]
	visited   map[string]struct{}
	direction Direction
	maxDepth  int
}

// NewDFS 创建DFS迭代器
func NewDFS[T comparable](g *graph.Graph[T], startID string, opts ...DFSOption[T]) (*DFS[T], error) {
	sn, err := g.GetNode(startID)
	if err != nil {
		return nil, err
	}

	dfs := &DFS[T]{
		graph:     g,
		stack:     []stackItem[T]{{node: sn, depth: 0}},
		visited:   make(map[string]struct{}),
		direction: Outgoing,
		maxDepth:  -1,
	}

	for _, opt := range opts {
		opt(dfs)
	}

	return dfs, nil
}

// 修改选项函数签名
func WithDirection[T comparable](d Direction) DFSOption[T] {
	return func(dfs *DFS[T]) {
		dfs.direction = d
	}
}

func WithMaxDepth[T comparable](depth int) DFSOption[T] {
	return func(dfs *DFS[T]) {
		dfs.maxDepth = depth
	}
}

// 核心方法实现
func (d *DFS[T]) HasNext() bool {
	return len(d.stack) > 0
}

// 获取当前遍历深度
func (d *DFS[T]) CurDepth() int {
	if len(d.stack) == 0 {
		return -1
	}
	return d.stack[len(d.stack)-1].depth
}

func (d *DFS[T]) Next() *graph.Node[T] {
	if !d.HasNext() {
		return nil
	}

	// 弹出当前节点及其深度
	currentItem := d.stack[len(d.stack)-1]
	d.stack = d.stack[:len(d.stack)-1]

	// 检查是否已访问
	if _, exists := d.visited[currentItem.node.ID]; exists {
		return d.Next()
	}

	// 标记已访问
	d.visited[currentItem.node.ID] = struct{}{}

	// 展开邻居节点（在深度限制内）
	if d.maxDepth < 0 || currentItem.depth < d.maxDepth {
		neighbors := d.getNeighbors(currentItem.node)
		for _, n := range neighbors {
			if _, visited := d.visited[n.ID]; !visited {
				d.stack = append(d.stack, stackItem[T]{
					node:  n,
					depth: currentItem.depth + 1, // 深度递增
				})
			}
		}
	}

	return currentItem.node
}

func (d *DFS[T]) Iterate(fn func(*graph.Node[T]) error) error {
	for d.HasNext() {
		node := d.Next()
		if node == nil {
			return fmt.Errorf("遇到空节点")
		}

		if err := fn(node); err != nil {
			return err
		}
	}
	return nil
}

// 获取邻居节点（核心逻辑）
func (d *DFS[T]) getNeighbors(n *graph.Node[T]) []*graph.Node[T] {
	var edges []*graph.Edge
	var err error

	switch d.direction {
	case Incoming:
		edges, err = d.graph.GetInEdges(n.ID)
	default:
		edges, err = d.graph.GetOutEdges(n.ID)
	}

	if err != nil || len(edges) == 0 {
		return nil
	}

	neighbors := make([]*graph.Node[T], 0, len(edges))
	for _, e := range edges {
		var neighborID string
		if d.direction == Incoming {
			neighborID = e.From
		} else {
			neighborID = e.To
		}

		if neighbor, err := d.graph.GetNode(neighborID); err == nil {
			neighbors = append(neighbors, neighbor)
		}
	}
	return neighbors
}
