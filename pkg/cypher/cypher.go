package cypher

import (
	"fmt"
	"grapher/internal/graph"
	"grapher/internal/traverse"
	"strconv"
)

// ExecuteQuery 执行查询并返回结果（基于DFS迭代器）
func ExecuteQuery[T comparable](q Query, g *graph.Graph[T]) ([]map[string]interface{}, error) {
	results := []map[string]interface{}{}
	sq := q.Root

	// 处理MATCH子句
	if len(sq.Reading) == 0 {
		return nil, fmt.Errorf("no MATCH clause found")
	}
	matchClause := sq.Reading[0]

	// 解析模式中的变量绑定
	var (
		startVar    Variable
		endVar      Variable
		edgePattern EdgePattern
	)

	// 提取模式中的节点和边信息
	for _, mp := range matchClause.Pattern {
		for _, elem := range mp.Elements {
			switch v := elem.(type) {
			case *NodePattern:
				if v.Variable != nil {
					if startVar == "" {
						startVar = *v.Variable
					} else {
						endVar = *v.Variable
					}
				}
			case *EdgePattern:
				edgePattern = *v
			}
		}
	}

	// 查找起始节点
	startNodes, err := findStartNodes(g, matchClause)
	if err != nil {
		return nil, err
	}

	// 遍历所有起始节点
	for _, startNode := range startNodes {
		// 设置深度范围
		minHops := 1 // Cypher默认至少1跳
		if edgePattern.MinHops != nil {
			minHops = *edgePattern.MinHops
		}
		maxHops := -1
		if edgePattern.MaxHops != nil {
			maxHops = *edgePattern.MaxHops
		}

		// 配置DFS参数
		opts := []traverse.DFSOption[T]{ // 根据泛型类型具体化
			traverse.WithDirection[T](convertDirection(edgePattern.Direction)),
			traverse.WithMaxDepth[T](maxHops),
		}

		// 创建DFS迭代器
		dfs, err := traverse.NewDFS(g, startNode.ID, opts...)
		if err != nil {
			return nil, err
		}

		// 收集结果
		seen := make(map[interface{}]bool)
		err = dfs.Iterate(func(n *graph.Node[T]) error {
			// 过滤跳数范围
			currentDepth := dfs.CurDepth()
			if currentDepth < minHops || (maxHops != -1 && currentDepth > maxHops) {
				return nil
			}

			// 结果去重
			if seen[n.Data] {
				return nil
			}
			seen[n.Data] = true

			// 构建结果项
			result := make(map[string]interface{})
			if startVar != "" {
				result[string(startVar)] = startNode.Data
			}
			if endVar != "" {
				result[string(endVar)] = n.Data
			}
			results = append(results, result)
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// 辅助函数 ---------------------------------------------------

// 转换方向枚举
func convertDirection(d EdgeDirection) traverse.Direction {
	switch d {
	case EdgeRight:
		return traverse.Outgoing
	case EdgeLeft:
		return traverse.Incoming
	default:
		return traverse.Outgoing
	}
}

// 查找起始节点
func findStartNodes[T comparable](g *graph.Graph[T], clause ReadingClause) ([]*graph.Node[T], error) {
	if len(clause.Pattern) == 0 {
		return nil, fmt.Errorf("no pattern found in MATCH clause")
	}

	// 获取第一个节点模式
	firstElem := clause.Pattern[0].Elements[0]
	nodePattern, ok := firstElem.(*NodePattern)
	if !ok {
		return nil, fmt.Errorf("expected NodePattern as first element")
	}

	return findNodesByPattern(g, *nodePattern)
}

func findNodesByPattern[T comparable](g *graph.Graph[T], np NodePattern) ([]*graph.Node[T], error) {
	allNodes := g.AllNodes()
	matched := make([]*graph.Node[T], 0)

	fmt.Printf("\n[DEBUG] Matching node pattern: Labels=%v, Properties=%v\n", np.Labels, np.Properties)

NODE_LOOP:
	for _, node := range allNodes {
		fmt.Printf("\n[DEBUG] Checking node %s:\n", node.ID)
		fmt.Printf("  - Labels: %v\n", node.Labels)
		fmt.Printf("  - Properties: %v\n", node.Data)

		// 检查标签
		if len(np.Labels) > 0 {
			if !hasAllLabels(node, np.Labels) {
				fmt.Printf("  ✗ Label mismatch\n")
				continue
			}
			fmt.Printf("  ✓ Labels matched\n")
		}

		// 检查属性
		for k, expr := range np.Properties {
			nodeVal, exists := node.Properties[k]
			if !exists {
				fmt.Printf("  ✗ Property '%s' not found\n", k)
				continue NODE_LOOP
			}

			switch v := expr.(type) {
			case StrLiteral:
				expected := string(v)
				actual := fmt.Sprint(nodeVal)
				if actual != expected {
					fmt.Printf("  ✗ Property '%s' value mismatch: expected '%s', got '%s'\n", k, expected, actual)
					continue NODE_LOOP
				}
				fmt.Printf("  ✓ Property '%s' matched ('%s')\n", k, expected)
			case IntegerLiteral:
				expected := int(v)
				actual, err := strconv.Atoi(fmt.Sprint(nodeVal))
				if err != nil || actual != expected {
					fmt.Printf("  ✗ Property '%s' value mismatch: expected %d, got %v\n", k, expected, nodeVal)
					continue NODE_LOOP
				}
				fmt.Printf("  ✓ Property '%s' matched (%d)\n", k, expected)
			default:
				fmt.Printf("  ✗ Unsupported property type: %T\n", expr)
				continue NODE_LOOP
			}
		}

		matched = append(matched, node)
		fmt.Printf("  ✓ Node matched\n")
	}

	fmt.Printf("\n[DEBUG] Total matched nodes: %d\n", len(matched))
	return matched, nil
}

// 标签检查
func hasAllLabels[T comparable](n *graph.Node[T], labels []string) bool {
	for _, l := range labels {
		if !contains(n.Labels, l) {
			return false
		}
	}
	return true
}

// 通用包含检查
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
