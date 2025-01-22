package graph

import (
	"encoding/json"
	"fmt"
	"os"
)

//--- 持久化操作 ---

// 序列化专用结构体
type graphDTO[T any] struct {
	Nodes []*Node[T] `json:"nodes"`
	Edges []*Edge    `json:"edges"`
}

// SaveToFile 保存图数据到文件
func (g *Graph[T]) SaveToFile(filename string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	dto := &graphDTO[T]{
		Nodes: make([]*Node[T], 0, len(g.nodes)),
		Edges: make([]*Edge, 0, len(g.out)*2),
	}

	for _, node := range g.nodes {
		dto.Nodes = append(dto.Nodes, node)
	}

	for _, edges := range g.out {
		for _, edge := range edges {
			dto.Edges = append(dto.Edges, edge)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(dto); err != nil {
		return fmt.Errorf("failed to encode graph: %w", err)
	}

	return nil
}

// LoadFromFile 从文件加载图数据
func (g *Graph[T]) LoadFromFile(filename string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var dto graphDTO[T]
	if err := json.NewDecoder(file).Decode(&dto); err != nil {
		return fmt.Errorf("failed to decode graph: %w", err)
	}

	// 清空现有数据
	g.nodes = make(map[string]*Node[T])
	g.out = make(map[string]map[string]*Edge)
	g.in = make(map[string]map[string]*Edge)

	// 加载节点
	for _, node := range dto.Nodes {
		if node.ID == "" {
			return fmt.Errorf("%w: empty node ID", ErrInvalidInput)
		}
		g.nodes[node.ID] = node
	}

	// 加载边并验证节点存在性
	for _, edge := range dto.Edges {
		if _, exists := g.nodes[edge.From]; !exists {
			return fmt.Errorf("%w: edge references missing node %s", ErrInvalidInput, edge.From)
		}
		if _, exists := g.nodes[edge.To]; !exists {
			return fmt.Errorf("%w: edge references missing node %s", ErrInvalidInput, edge.To)
		}
		g.addEdgeToIndex(edge.From, edge.To, edge)
	}

	return nil
}
