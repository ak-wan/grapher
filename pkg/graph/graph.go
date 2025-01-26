package graph

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNodeExists   = errors.New("node already exists")
	ErrNodeNotFound = errors.New("node not found")
	ErrEdgeExists   = errors.New("edge already exists")
	ErrEdgeNotFound = errors.New("edge not found")
	ErrInvalidInput = errors.New("invalid input data")
)

// Node 表示图节点，支持泛型属性值
type Node[T any] struct {
	ID         string       `json:"id"`
	Labels     []string     `json:"labels"`
	Properties map[string]T `json:"properties"`
}

// Edge 表示有向带权边
type Edge struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Weight float64 `json:"weight"`
}

// Graph 并发安全的有向带权图
type Graph[T any] struct {
	mu    sync.RWMutex
	nodes map[string]*Node[T]         // 节点存储
	in    map[string]map[string]*Edge // 入边索引：to -> from -> Edge
	out   map[string]map[string]*Edge // 出边索引：from -> to -> Edge
}

// New 创建新图实例
func New[T any]() *Graph[T] {
	return &Graph[T]{
		nodes: make(map[string]*Node[T]),
		in:    make(map[string]map[string]*Edge),
		out:   make(map[string]map[string]*Edge),
	}
}

// --- 节点操作 ---

// AddNode 添加节点（带初始化属性）
func (g *Graph[T]) AddNode(id string, props map[string]T) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if id == "" {
		return ErrInvalidInput
	}

	if _, exists := g.nodes[id]; exists {
		return fmt.Errorf("%w: %s", ErrNodeExists, id)
	}

	g.nodes[id] = &Node[T]{
		ID:         id,
		Properties: props, // 属性直接存储
	}
	return nil
}

// UpdateNodeProps 更新节点属性
func (g *Graph[T]) UpdateNodeProps(id string, props map[string]T) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	node, exists := g.nodes[id]
	if !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, id)
	}

	for k, v := range props {
		node.Properties[k] = v
	}
	return nil
}

// RemoveNode 删除节点及关联边
func (g *Graph[T]) RemoveNode(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[id]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, id)
	}

	// 删除出边
	for to := range g.out[id] {
		delete(g.in[to], id)
		if len(g.in[to]) == 0 {
			delete(g.in, to)
		}
	}
	delete(g.out, id)

	// 删除入边
	for from := range g.in[id] {
		delete(g.out[from], id)
		if len(g.out[from]) == 0 {
			delete(g.out, from)
		}
	}
	delete(g.in, id)

	delete(g.nodes, id)
	return nil
}

// --- 边操作 ---

// AddEdge 添加带权边
func (g *Graph[T]) AddEdge(from, to string, weight float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if from == "" || to == "" {
		return ErrInvalidInput
	}

	if _, exists := g.nodes[from]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, from)
	}
	if _, exists := g.nodes[to]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, to)
	}

	if _, exists := g.out[from][to]; exists {
		return fmt.Errorf("%w: %s->%s", ErrEdgeExists, from, to)
	}

	g.addEdgeToIndex(from, to, &Edge{From: from, To: to, Weight: weight})
	return nil
}

// UpdateEdge 更新边权重
func (g *Graph[T]) UpdateEdge(from, to string, weight float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	edge, exists := g.out[from][to]
	if !exists {
		return fmt.Errorf("%w: %s->%s", ErrEdgeNotFound, from, to)
	}

	edge.Weight = weight
	return nil
}

// GetEdge 获取边
func (g *Graph[T]) GetEdge(from, to string) (*Edge, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if edges, exists := g.out[from]; exists {
		if edge, exists := edges[to]; exists {
			return edge, nil
		}
	}
	return nil, fmt.Errorf("%w: %s->%s", ErrEdgeNotFound, from, to)
}

// RemoveEdge 移除边
func (g *Graph[T]) RemoveEdge(from, to string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.out[from][to]; !exists {
		return fmt.Errorf("%w: %s->%s", ErrEdgeNotFound, from, to)
	}

	delete(g.out[from], to)
	if len(g.out[from]) == 0 {
		delete(g.out, from)
	}

	delete(g.in[to], from)
	if len(g.in[to]) == 0 {
		delete(g.in, to)
	}

	return nil
}

// --- 查询操作 ---

// GetNode 获取节点
func (g *Graph[T]) GetNode(id string) (*Node[T], error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, exists := g.nodes[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, id)
	}
	return node, nil
}

// AllNodes 返回全部节点
func (g *Graph[T]) AllNodes() []*Node[T] {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]*Node[T], 0, len(g.nodes))
	for _, node := range g.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNodesByProp 根据属性查找节点
func (g *Graph[T]) GetNodesByProp(key string, value T) []*Node[T] {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]*Node[T], 0)
	for _, node := range g.nodes {
		if v, exists := node.Properties[key]; exists && any(v) == any(value) {
			result = append(result, node)
		}
	}
	return result
}

// GetOutEdges 获取出边
func (g *Graph[T]) GetOutEdges(from string) ([]*Edge, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.nodes[from]; !exists {
		return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, from)
	}

	edges := make([]*Edge, 0, len(g.out[from]))
	for _, e := range g.out[from] {
		edges = append(edges, e)
	}
	return edges, nil
}

// 添加反向索引操作封装
func (g *Graph[T]) addEdgeToIndex(from, to string, edge *Edge) {
	if _, exists := g.out[from]; !exists {
		g.out[from] = make(map[string]*Edge)
	}
	g.out[from][to] = edge

	if _, exists := g.in[to]; !exists {
		g.in[to] = make(map[string]*Edge)
	}
	g.in[to][from] = edge
}

// GetInEdges 获取入边
func (g *Graph[T]) GetInEdges(to string) ([]*Edge, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.nodes[to]; !exists {
		return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, to)
	}

	edges := make([]*Edge, 0, len(g.in[to]))
	for _, e := range g.in[to] {
		edges = append(edges, e)
	}
	return edges, nil
}
