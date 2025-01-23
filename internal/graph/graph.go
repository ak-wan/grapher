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

// Node 表示图节点，支持泛型数据
type Node[T any] struct {
	ID         string `json:"id"`
	Data       T      `json:"data"`
	Labels     []string
	Properties map[string]interface{}
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
	in    map[string]map[string]*Edge // 入边索引：to -> from -> Edge (新增反向索引)
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

// AddNode 在图上添加节点
func (g *Graph[T]) AddNode(id string, data T) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if id == "" {
		return ErrInvalidInput
	}

	if _, exists := g.nodes[id]; exists {
		return fmt.Errorf("%w: %s", ErrNodeExists, id)
	}

	g.nodes[id] = &Node[T]{ID: id, Data: data}
	return nil
}

// RemoveNode 删除节点及关联边
func (g *Graph[T]) RemoveNode(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[id]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, id)
	}

	// 删除所有出边
	for to := range g.out[id] {
		delete(g.in[to], id)
		if len(g.in[to]) == 0 {
			delete(g.in, to)
		}
	}
	delete(g.out, id)

	// 删除所有入边
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

//--- 边操作 ---

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

//--- 查询操作 ---

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

// GetOutEdges 获取节点的出边
func (g *Graph[T]) GetOutEdges(from string) ([]*Edge, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.nodes[from]; !exists {
		return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, from)
	}

	edges := make([]*Edge, 0, len(g.out[from]))
	for _, edge := range g.out[from] {
		edges = append(edges, edge)
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

// 添加获取入边方法
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
