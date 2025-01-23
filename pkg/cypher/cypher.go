package cypher

import (
	"fmt"
	"grapher/internal/graph"
	"strconv"
)

// ExecuteQuery 执行 Cypher 查询并返回结果（支持泛型图结构）
func ExecuteQuery[T comparable](q Query, g *graph.Graph[T]) ([]map[string]interface{}, error) {
	results := []map[string]interface{}{}

	// 获取查询的根节点（单个查询）
	sq := q.Root

	// 处理所有 MATCH 子句（目前只处理第一个 MATCH）
	if len(sq.Reading) == 0 {
		return nil, fmt.Errorf("no MATCH clause found")
	}
	matchClause := sq.Reading[0]

	// 提取匹配模式中的变量绑定
	var startVar, endVar Variable
	var edgePattern EdgePattern
	for _, mp := range matchClause.Pattern {
		for _, elem := range mp.Elements {
			switch v := elem.(type) {
			case *NodePattern: // 指针类型
				if v.Variable != nil {
					if startVar == "" {
						startVar = *v.Variable
					} else {
						endVar = *v.Variable
					}
				}
			case *EdgePattern:
				edgePattern = *v
			default:
				return nil, fmt.Errorf("unsupported pattern element type: %T", elem)
			}
		}
	}

	// 查找起始节点
	nodePattern, ok := matchClause.Pattern[0].Elements[0].(*NodePattern)
	if !ok {
		return nil, fmt.Errorf("expected NodePattern but got %T", matchClause.Pattern[0].Elements[0])
	}
	startNodes, err := findNodesByPattern(g, *nodePattern)
	if err != nil {
		return nil, err
	}

	// 遍历所有起始节点
	for _, startNode := range startNodes {
		paths := traversePaths(g, startNode, edgePattern)

		// 收集结果并去重
		seen := make(map[interface{}]bool)
		for _, path := range paths {
			endNode := path.End()

			// 生成结果键
			resultKey := struct {
				Start interface{}
				End   interface{}
			}{startNode.Data, endNode.Data}

			if seen[resultKey] {
				continue
			}
			seen[resultKey] = true

			result := make(map[string]interface{})
			if startVar != "" {
				result[string(startVar)] = startNode.Data
			}
			if endVar != "" {
				result[string(endVar)] = endNode.Data
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// 辅助结构体（使用泛型）
type Path[T comparable] struct {
	Current  *graph.Node[T]
	Previous *Path[T]
	Depth    int
}

func (p *Path[T]) End() *graph.Node[T] {
	if p.Previous == nil {
		return p.Current
	}
	return p.Previous.End()
}

// 查找匹配节点（泛型版本）
func findNodesByPattern[T comparable](g *graph.Graph[T], np NodePattern) ([]*graph.Node[T], error) {
	nodes := g.AllNodes()
	filtered := make([]*graph.Node[T], 0)

	for _, node := range nodes {
		// 检查标签
		if len(np.Labels) > 0 && !hasAllLabels(node, np.Labels) {
			continue
		}

		// 检查属性
		if match, err := matchProperties(node, np.Properties); err != nil {
			return nil, err
		} else if !match {
			continue
		}

		filtered = append(filtered, node)
	}

	return filtered, nil
}

// 路径遍历核心逻辑（泛型版本）
func traversePaths[T comparable](g *graph.Graph[T], start *graph.Node[T], ep EdgePattern) []*Path[T] {
	queue := []*Path[T]{{Current: start, Depth: 0}}
	results := []*Path[T]{}

	// 处理跳数范围
	minHops := 1
	if ep.MinHops != nil {
		minHops = *ep.MinHops
	}

	maxHops := -1 // 无限
	if ep.MaxHops != nil {
		maxHops = *ep.MaxHops
	}

	for len(queue) > 0 {
		currentPath := queue[0]
		queue = queue[1:]

		// 检查是否满足终止条件
		if currentPath.Depth >= minHops && (maxHops == -1 || currentPath.Depth <= maxHops) {
			results = append(results, currentPath)
		}

		// 超过最大深度则停止
		if maxHops != -1 && currentPath.Depth >= maxHops {
			continue
		}

		// 获取当前节点的出边（根据方向）
		var edges []*graph.Edge
		switch ep.Direction {
		case EdgeRight:
			edges, _ = g.GetOutEdges(currentPath.Current.ID)
		case EdgeLeft:
			edges, _ = g.GetInEdges(currentPath.Current.ID)
		default:
			outEdges, err := g.GetOutEdges(currentPath.Current.ID)
			if err != nil {
				continue
			}
			inEdges, err := g.GetInEdges(currentPath.Current.ID)
			if err != nil {
				continue
			}
			edges = append(outEdges, inEdges...)
		}

		// 遍历边
		for _, edge := range edges {
			var nextNode *graph.Node[T]
			var err error

			if edge.From == currentPath.Current.ID {
				nextNode, err = g.GetNode(edge.To)
				if err != nil {
					continue
				}
			} else {
				nextNode, err = g.GetNode(edge.From)
				if err != nil {
					continue
				}
			}

			// 防止循环（检查当前路径是否包含该节点）
			if pathContains(currentPath, nextNode) {
				continue
			}

			queue = append(queue, &Path[T]{
				Current:  nextNode,
				Previous: currentPath,
				Depth:    currentPath.Depth + 1,
			})
		}
	}

	return results
}

// 辅助函数 ---------------------------------------------------

// 检查路径是否包含节点
func pathContains[T comparable](path *Path[T], node *graph.Node[T]) bool {
	for p := path; p != nil; p = p.Previous {
		if p.Current.ID == node.ID {
			return true
		}
	}
	return false
}

// 属性匹配（支持多种表达式类型）
func matchProperties[T comparable](n *graph.Node[T], props map[string]Expr) (bool, error) {
	for key, expr := range props {
		// 获取节点属性值
		nodeVal, ok := n.Properties[key]
		if !ok {
			return false, nil
		}

		// 表达式求值
		switch v := expr.(type) {
		case StrLiteral:
			if nodeVal != string(v) {
				return false, nil
			}
		case IntegerLiteral:
			num, err := strconv.Atoi(fmt.Sprint(nodeVal))
			if err != nil || num != int(v) {
				return false, nil
			}
		case Variable:
			// 变量需要从上下文中获取值（此处简化处理）
			return false, fmt.Errorf("variable properties not supported yet")
		default:
			return false, fmt.Errorf("unsupported property type: %T", expr)
		}
	}
	return true, nil
}

// 标签检查（假设graph.Node有Labels字段）
func hasAllLabels[T comparable](n *graph.Node[T], labels []string) bool {
	for _, l := range labels {
		if !contains(n.Labels, l) {
			return false
		}
	}
	return true
}

// 通用辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
