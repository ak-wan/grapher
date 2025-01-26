package cypher

import (
	"fmt"
	"grapher/internal/graph"
	"grapher/internal/traverse"
	"reflect"
	"strconv"
)

// ExecuteQuery 支持范围过滤的查询执行（完整版）
func ExecuteQuery[T comparable](q Query, g *graph.Graph[T]) ([]map[string]interface{}, error) {
	results := []map[string]interface{}{}
	if len(q.Root.Reading) == 0 {
		return nil, fmt.Errorf("no MATCH clause found")
	}
	matchClause := q.Root.Reading[0]

	// 确保只处理单个模式
	if len(matchClause.Pattern) != 1 {
		return nil, fmt.Errorf("only single pattern is supported")
	}

	// 解析模式结构 (start)-[edge]->(end)
	var (
		edge         EdgePattern
		startPattern *NodePattern
		endPattern   *NodePattern
	)

	// 提取模式中的元素
	for _, mp := range matchClause.Pattern {
		if len(mp.Elements) != 3 {
			return nil, fmt.Errorf("invalid pattern structure, expected (start)-[...]->(end)")
		}

		// 解析起始节点
		if np, ok := mp.Elements[0].(*NodePattern); ok {
			startPattern = np
		} else {
			return nil, fmt.Errorf("first element must be node pattern")
		}

		// 解析边模式
		if ep, ok := mp.Elements[1].(*EdgePattern); ok {
			edge = *ep
		} else {
			return nil, fmt.Errorf("second element must be edge pattern")
		}

		// 解析终止节点
		if np, ok := mp.Elements[2].(*NodePattern); ok {
			endPattern = np
		} else {
			return nil, fmt.Errorf("third element must be node pattern")
		}
	}

	// 查找起始节点
	startNodes, err := findStartNodes(g, matchClause)
	if err != nil {
		return nil, fmt.Errorf("start node error: %w", err)
	}

	// 遍历所有起始节点
	for _, startNode := range startNodes {
		endFilter := nodeMatchesPattern[T](endPattern)

		opts := []traverse.DFSOption[T]{
			traverse.WithDirection[T](convertDirection(edge.Direction)),
			traverse.WithRangeFilter[T](
				func(n *graph.Node[T]) bool { // 起始节点已经过筛选
					return nodeMatchesPattern[T](startPattern)(n)
				},
				endFilter,
			),
		}

		// 初始化DFS遍历器
		dfs, err := traverse.NewDFS(g, startNode.ID, opts...)
		if err != nil {
			return nil, fmt.Errorf("DFS init failed: %w", err)
		}

		// 收集结果
		dfs.Iterate(func(n *graph.Node[T]) error {
			// 构建结果记录
			result := map[string]interface{}{
				"ID":         n.ID,
				"Properties": n.Properties,
			}
			results = append(results, result)
			return nil
		})

	}

	return results, nil
}

// 辅助函数 ---------------------------------------------------

func convertDirection(d EdgeDirection) traverse.Direction {
	switch d {
	case EdgeLeft:
		return traverse.Incoming
	default:
		return traverse.Outgoing
	}
}

func findStartNodes[T comparable](g *graph.Graph[T], clause ReadingClause) ([]*graph.Node[T], error) {
	if len(clause.Pattern) == 0 {
		return nil, fmt.Errorf("empty pattern")
	}

	firstElem := clause.Pattern[0].Elements[0]
	np, ok := firstElem.(*NodePattern)
	if !ok {
		return nil, fmt.Errorf("first element must be node pattern")
	}

	fmt.Println("[DEBUG] Searching for start nodes\n", np.Properties)
	return findNodesByPattern(g, *np)
}

func findNodesByPattern[T comparable](g *graph.Graph[T], np NodePattern) ([]*graph.Node[T], error) {
	fmt.Printf("[DEBUG] Searching for nodes matching: %+v\n", np)
	matched := make([]*graph.Node[T], 0)
	matcher := nodeMatchesPattern[T](&np)
	for _, node := range g.AllNodes() {
		if !matcher(node) {
			continue
		}
		matched = append(matched, node)
	}
	return matched, nil
}

func nodeMatchesPattern[T comparable](np *NodePattern) func(*graph.Node[T]) bool {
	if np == nil {
		return func(*graph.Node[T]) bool { return true }
	}

	return func(node *graph.Node[T]) bool {
		// 属性匹配
		for key, expr := range np.Properties {
			nodeVal, exists := node.Properties[key]
			if !exists {
				return false
			}

			switch v := expr.(type) {
			case StrLiteral:
				if fmt.Sprint(nodeVal) != string(v) {
					return false
				}
			case IntegerLiteral:
				expected := int(v)
				// 改进类型处理逻辑
				val := reflect.ValueOf(nodeVal)
				switch val.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if int(val.Int()) != expected {
						return false
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					if int(val.Uint()) != expected {
						return false
					}
				case reflect.Float32, reflect.Float64:
					if int(val.Float()) != expected {
						return false
					}
				case reflect.String:
					parsed, err := strconv.Atoi(val.String())
					if err != nil || parsed != expected {
						return false
					}
				default:
					return false
				}
			default:
				return false
			}
		}
		return true
	}
}
