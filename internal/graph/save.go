package graph

import (
	"encoding/json"
	"fmt"
	"os"
)

//--- 持久化操作 ---

// 序列化专用结构体（避免直接暴露内部结构）
type graphDTO[T any] struct {
	Nodes []nodeDTO[T] `json:"nodes"`
	Edges []edgeDTO    `json:"edges"`
}

type nodeDTO[T any] struct {
	ID         string       `json:"id"`
	Labels     []string     `json:"labels"`
	Properties map[string]T `json:"properties"`
}

type edgeDTO struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Weight float64 `json:"weight"`
}

// SaveToFile 保存图数据到文件
func (g *Graph[T]) SaveToFile(filename string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// 构建DTO结构
	dto := graphDTO[T]{
		Nodes: make([]nodeDTO[T], 0, len(g.nodes)),
		Edges: make([]edgeDTO, 0, len(g.out)*2),
	}

	// 转换节点
	for _, node := range g.nodes {
		dto.Nodes = append(dto.Nodes, nodeDTO[T]{
			ID:         node.ID,
			Labels:     node.Labels,
			Properties: node.Properties,
		})
	}

	// 转换边
	for _, edges := range g.out {
		for _, edge := range edges {
			dto.Edges = append(dto.Edges, edgeDTO{
				From:   edge.From,
				To:     edge.To,
				Weight: edge.Weight,
			})
		}
	}

	// 写入文件
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

	// 读取文件
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 解析DTO
	var dto graphDTO[T]
	if err := json.NewDecoder(file).Decode(&dto); err != nil {
		return fmt.Errorf("failed to decode graph: %w", err)
	}

	// 清空现有数据
	g.nodes = make(map[string]*Node[T])
	g.in = make(map[string]map[string]*Edge)
	g.out = make(map[string]map[string]*Edge)

	// 加载节点
	nodeIDMap := make(map[string]struct{})
	for _, node := range dto.Nodes {
		if node.ID == "" {
			return fmt.Errorf("%w: empty node ID", ErrInvalidInput)
		}
		if _, exists := nodeIDMap[node.ID]; exists {
			return fmt.Errorf("%w: duplicate node ID %s", ErrInvalidInput, node.ID)
		}
		nodeIDMap[node.ID] = struct{}{}

		g.nodes[node.ID] = &Node[T]{
			ID:         node.ID,
			Labels:     node.Labels,
			Properties: node.Properties,
		}
	}

	// 加载边
	for _, edge := range dto.Edges {
		// 验证节点存在性
		if _, exists := g.nodes[edge.From]; !exists {
			return fmt.Errorf("%w: edge references missing node %s", ErrInvalidInput, edge.From)
		}
		if _, exists := g.nodes[edge.To]; !exists {
			return fmt.Errorf("%w: edge references missing node %s", ErrInvalidInput, edge.To)
		}

		// 使用标准方法添加边（维护索引）
		if err := g.addEdgeInternal(edge.From, edge.To, edge.Weight); err != nil {
			return fmt.Errorf("failed to add edge %s->%s: %w", edge.From, edge.To, err)
		}
	}

	return nil
}

// 内部添加边方法（无锁，需在已加锁环境下调用）
func (g *Graph[T]) addEdgeInternal(from, to string, weight float64) error {
	// 初始化索引
	if _, exists := g.out[from]; !exists {
		g.out[from] = make(map[string]*Edge)
	}
	if _, exists := g.in[to]; !exists {
		g.in[to] = make(map[string]*Edge)
	}

	// 检查边是否已存在
	if _, exists := g.out[from][to]; exists {
		return fmt.Errorf("%w: %s->%s", ErrEdgeExists, from, to)
	}

	// 创建边对象
	edge := &Edge{
		From:   from,
		To:     to,
		Weight: weight,
	}

	// 更新索引
	g.out[from][to] = edge
	g.in[to][from] = edge
	return nil
}
