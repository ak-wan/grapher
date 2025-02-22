package ast

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Parser 表示 Cypher 解析器
type Parser struct {
	s *bufScanner
}

// NewParser 返回一个新的 Parser 实例
func NewParser(r io.Reader) *Parser {
	return &Parser{s: newBufScanner(r)}
}

// ParseQuery 解析 Cypher 字符串并返回 Query 抽象语法树对象
func (p *Parser) ParseQuery() (*SingleQuery, error) {
	for {
		// 扫描到文件末尾时返回
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok == EOF {
			return nil, nil
		} else if tok == SEMICOLON {
			continue // 跳过分号
		} else {
			p.Unscan() // 回退当前标记
			return p.ParseSingleQuery()
		}
	}
}

// ParseSingleQuery 解析单个查询语句（如 MATCH...RETURN...）
func (p *Parser) ParseSingleQuery() (*SingleQuery, error) {
	sq := &SingleQuery{}

	// 解析所有 READING 子句（MATCH/OPTIONAL MATCH）
	for {
		tok, _, _ := p.ScanIgnoreWhitespace()
		if tok != MATCH && tok != OPTIONAL {
			p.Unscan()
			break
		}
		p.Unscan()

		rc, err := p.ScanReadingClause()
		if err != nil {
			return nil, err
		}
		sq.Reading = append(sq.Reading, *rc)
	}

	// RETURN 子句是强制性的
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != RETURN {
		return nil, newParseError(tokstr(tok, lit), []string{"RETURN"}, pos)
	}

	// 处理 DISTINCT 修饰符
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == DISTINCT {
		sq.Distinct = true
	} else {
		p.Unscan()
	}

	// 解析 RETURN 的返回项列表
	for {
		// 解析表达式（如 A, n）
		expr, err := p.ScanExpression()
		if err != nil {
			return nil, err
		}
		sq.ReturnItems = append(sq.ReturnItems, expr)

		// 检查是否有更多返回项
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != COMMA {
			p.Unscan()
			break
		}
	}

	// 解析可选的 ORDER BY 子句
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == ORDER {
		if tokBy, pos, lit := p.ScanIgnoreWhitespace(); tokBy != BY {
			return nil, newParseError(tokstr(tokBy, lit), []string{"BY"}, pos)
		}

		// 解析排序项列表
		for {
			expr, err := p.ScanExpression()
			if err != nil {
				return nil, err
			}

			// 默认升序
			dir := Ascending
			if tokDir, _, _ := p.ScanIgnoreWhitespace(); tokDir == DESC || tokDir == DESCENDING {
				dir = Descending
			} else if tokDir == ASC || tokDir == ASCENDING {
				// 已经是默认值，不需要处理
			} else {
				p.Unscan()
			}

			sq.Order = append(sq.Order, OrderBy{
				Dir:  dir,
				Item: expr,
			})

			// 检查是否有更多排序项
			if tok, _, _ := p.ScanIgnoreWhitespace(); tok != COMMA {
				p.Unscan()
				break
			}
		}
	} else {
		p.Unscan()
	}

	// 解析可选的 SKIP
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == SKIP {
		expr, err := p.ScanExpression()
		if err != nil {
			return nil, err
		}
		sq.Skip = &expr
	} else {
		p.Unscan()
	}

	// 解析可选的 LIMIT
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == LIMIT {
		expr, err := p.ScanExpression()
		if err != nil {
			return nil, err
		}
		sq.Limit = &expr
	} else {
		p.Unscan()
	}

	return sq, nil
}

// ScanReadingClause 扫描读取子句（MATCH/OPTIONAL MATCH）
func (p *Parser) ScanReadingClause() (*ReadingClause, error) {
	rc := &ReadingClause{}

	// 检查是否是 OPTIONAL MATCH
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == OPTIONAL {
		rc.OptionalMatch = true
	} else {
		p.Unscan()
	}

	// MATCH 是必须的关键字
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != MATCH {
		return nil, newParseError(tokstr(tok, lit), []string{"MATCH"}, pos)
	}

	// 解析匹配模式列表
	for {
		mp, err := p.ScanMatchPattern()
		if err != nil {
			return nil, err
		}
		rc.Pattern = append(rc.Pattern, *mp)

		// 检查是否有更多模式
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != COMMA {
			p.Unscan()
			break
		}
	}

	// 处理可选的 WHERE 条件
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == WHERE {
		exp, err := p.ScanExpression()
		if err != nil {
			return nil, err
		}
		rc.Where = &exp
	} else {
		p.Unscan()
	}

	return rc, nil
}

// ScanMatchPattern 扫描匹配模式
func (p *Parser) ScanMatchPattern() (*MatchPattern, error) {
	mp := &MatchPattern{}

	// 解析模式变量赋值（如 path = (a)-[...]->(b)）
	if tok, _, lit := p.ScanIgnoreWhitespace(); tok == IDENT {
		// 需要等号标识符
		if tok1, pos, lit1 := p.ScanIgnoreWhitespace(); tok1 != EQ {
			return nil, newParseError(tokstr(tok1, lit1), []string{"="}, pos)
		}

		v := Variable(lit)
		mp.Variable = &v
	} else {
		p.Unscan()
	}

	// 扫描模式元素
	elems, err := p.ScanPatternElements()
	if err != nil {
		return nil, err
	}
	mp.Elements = elems

	return mp, nil
}

// ScanPatternElements 解析模式链（如 (A)-[rel]->(B)-[rel2]->(C)）
func (p *Parser) ScanPatternElements() ([]PatternElement, error) {
	var elements []PatternElement

	// 解析第一个节点
	node, err := p.ScanNodePattern()
	if err != nil || node == nil {
		return nil, fmt.Errorf("expected node pattern")
	}
	elements = append(elements, node)

	// 循环解析关系-节点对
	for {
		// 检查是否有关系模式
		edge, err := p.ScanEdgePattern()
		if err != nil {
			return nil, err
		} else if edge == nil {
			break // 无更多关系模式
		}
		elements = append(elements, edge)

		// 解析下一个节点
		node, err := p.ScanNodePattern()
		if err != nil {
			return nil, err
		} else if node == nil {
			return nil, fmt.Errorf("expected node after relationship")
		}
		elements = append(elements, node)
	}

	return elements, nil
}

// ScanNodePattern 扫描节点模式（如 (a:Person {name: 'Alice'}））
func (p *Parser) ScanNodePattern() (*NodePattern, error) {
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok != LPAREN {
		p.Unscan()
		return nil, nil
	}
	var validNode bool
	var node NodePattern

	// 解析变量名
	if tok, _, lit := p.ScanIgnoreWhitespace(); tok == IDENT {
		v := Variable(lit)
		node.Variable = &v
		validNode = true
	} else {
		p.Unscan()
	}

	// 解析标签列表
	for {
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok == COLON {
			if tok1, pos, lit := p.ScanIgnoreWhitespace(); tok1 == IDENT {
				node.Labels = append(node.Labels, lit)
				validNode = true
			} else {
				return nil, newParseError(tokstr(tok, lit), []string{"Label Identifier"}, pos)
			}
		} else {
			p.Unscan()
			break
		}
	}

	// 解析属性
	props, err := p.ScanProperties()
	if err != nil {
		return nil, err
	} else if props != nil {
		node.Properties = *props
		validNode = true
	}

	// 检查闭合括号
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok == RPAREN {
		return &node, nil
	} else if validNode && tok != RPAREN {
		return nil, newParseError(tokstr(tok, lit), []string{")"}, pos)
	}

	p.Unscan()
	p.Unscan() // 回退到起始括号
	return nil, nil
}

// ScanEdgePattern 扫描边模式（如 -[r:KNOWS {since: 2010}]->）
func (p *Parser) ScanEdgePattern() (*EdgePattern, error) {
	ep := &EdgePattern{
		Direction: EdgeUndefined,
	}

	// 扫描起始符号（- 或 <-）
	tok1, _, _ := p.ScanIgnoreWhitespace()
	switch tok1 {
	case SUB:
		// 处理右箭头逻辑...

		tok2, pos2, lit2 := p.ScanIgnoreWhitespace()
		switch tok2 {
		case GT: // -> 右箭头
			ep.Direction = EdgeRight
		case REL_RANGE: // [*...]
			// 处理范围并确保闭合 ]
			if err := p.parseRelRange(ep, lit2); err != nil {
				return nil, err
			}
			// 解析后续箭头
			tok3, pos3, lit3 := p.ScanIgnoreWhitespace()
			if tok3 == EDGE_RIGHT {
				ep.Direction = EdgeRight
			} else {
				return nil, newParseError(tokstr(tok3, lit3), []string{"->"}, pos3)
			}
		case LBRACKET: // -[...]
			p.Unscan() // 回退 [ 以进入 parseEdgeDetails
			ep.Direction = EdgeRight
			if err := p.parseEdgeDetails(ep); err != nil {
				return nil, err
			}
			// 确保消费闭合的 ]
			if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != RBRACKET {
				return nil, newParseError(tokstr(tok, lit), []string{"]"}, pos)
			}
			// 处理箭头
			tok3, pos3, lit3 := p.ScanIgnoreWhitespace()
			if tok3 == SUB {
				tok4, pos4, lit4 := p.ScanIgnoreWhitespace()
				if tok4 == GT {
					ep.Direction = EdgeRight
				} else {
					return nil, newParseError(tokstr(tok4, lit4), []string{">"}, pos4)
				}
			} else {
				return nil, newParseError(tokstr(tok3, lit3), []string{"-"}, pos3)
			}
		default:
			return nil, newParseError(tokstr(tok2, lit2), []string{">", "[*"}, pos2)
		}
	case LT:
		// 处理左箭头逻辑...
	default:
		p.Unscan()
		return nil, nil
	}

	fmt.Printf("Parsed Edge: Variable=%v, Types=%v, Direction=%v, Min=%v, Max=%v\n", ep.Variable, ep.RelTypes, ep.Direction, ep.MinHops, ep.MaxHops)
	return ep, nil
}

// parseEdgeDetails 解析方括号内的关系详情
func (p *Parser) parseEdgeDetails(ep *EdgePattern) error {
	// 跳过 [
	for {
		tok, pos, lit := p.ScanIgnoreWhitespace()
		switch tok {
		case IDENT: // 变量名（如 rel）
			v := lit
			ep.Variable = &v
		case COLON: // 类型定义（如 :KNOWS）
			typeTok, pos, lit := p.ScanIgnoreWhitespace()
			if typeTok != IDENT {
				return newParseError(tokstr(typeTok, lit), []string{"relationship type"}, pos)
			}
			ep.RelTypes = append(ep.RelTypes, lit)
		case MUL: // 可变长度路径（如 *1..5）
			if err := p.parseRelRange(ep, lit); err != nil {
				return err
			}
		case LBRACE: // 属性（如 {prop: 'value'}）
			p.Unscan()
			props, err := p.ScanProperties()
			if err != nil {
				return err
			}
			ep.Properties = *props
		case RBRACKET: // 结束 ]
			return nil
		default:
			return newParseError(tokstr(tok, lit), []string{"identifier", "*", "}"}, pos)
		}
	}
}

// parseRelRange 解析关系范围（如 [*1..5]）
func (p *Parser) parseRelRange(ep *EdgePattern, lit string) error {
	rangeStr := strings.TrimPrefix(lit, "[*")
	rangeStr = strings.TrimSuffix(rangeStr, "]")

	parts := strings.Split(rangeStr, "..")
	if len(parts) == 0 {
		ep.MinHops = new(int) // 默认 0
		ep.MaxHops = new(int) // 默认 -1（无限）
		return nil
	}

	// 解析起始值
	if parts[0] != "" {
		start, _ := strconv.Atoi(parts[0])
		ep.MinHops = &start
	}

	// 解析结束值
	if len(parts) > 1 && parts[1] != "" {
		end, _ := strconv.Atoi(parts[1])
		ep.MaxHops = &end
	}

	return nil
}

// ScanExpression 扫描表达式（基础实现）
func (p *Parser) ScanExpression() (Expr, error) {
	tok, pos, lit := p.ScanIgnoreWhitespace()
	switch tok {
	case IDENT:
		return Variable(lit), nil
	case STRING:
		return StrLiteral(lit), nil
	case INTEGER:
		num, _ := strconv.Atoi(lit)
		return IntegerLiteral(num), nil
	default:
		return nil, newParseError(tokstr(tok, lit), []string{"identifier", "literal"}, pos)
	}
}

// ScanProperties 扫描属性键值对
func (p *Parser) ScanProperties() (*map[string]Expr, error) {
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok != LBRACE {
		p.Unscan()
		return nil, nil
	}

	props := make(map[string]Expr)
	for {
		// 键
		tokKey, pos, lit := p.ScanIgnoreWhitespace()
		if tokKey != IDENT {
			return nil, newParseError(tokstr(tokKey, lit), []string{"identifier"}, pos)
		}
		key := lit

		// 冒号
		if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != COLON {
			return nil, newParseError(tokstr(tok, lit), []string{":"}, pos)
		}

		// 值
		expr, err := p.ScanExpression()
		if err != nil {
			return nil, err
		}
		props[key] = expr

		// 逗号或结束符
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != COMMA {
			p.Unscan()
			break
		}
	}

	// 闭合大括号
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != RBRACE {
		return nil, newParseError(tokstr(tok, lit), []string{"}"}, pos)
	}

	return &props, nil
}

// Scan 返回下一个标记
func (p *Parser) Scan() (tok Token, pos Pos, lit string) { return p.s.Scan() }

// ScanIgnoreWhitespace 扫描下一个非空白标记
func (p *Parser) ScanIgnoreWhitespace() (tok Token, pos Pos, lit string) {
	for {
		tok, pos, lit = p.Scan()
		if tok == WS || tok == COMMENT {
			continue
		}
		return
	}
}

// Unscan 回退上一个扫描的标记
func (p *Parser) Unscan() { p.s.Unscan() }

// ParseError 表示解析过程中发生的错误
type ParseError struct {
	Message  string
	Found    string
	Expected []string
	Pos      Pos
}

// newParseError 创建新的解析错误实例
func newParseError(found string, expected []string, pos Pos) *ParseError {
	return &ParseError{Found: found, Expected: expected, Pos: pos}
}

// Error 返回错误信息字符串
func (e *ParseError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s at line %d, column %d", e.Message, e.Pos.Line, e.Pos.Column)
	}
	return fmt.Sprintf("Parse error. Found %s, expected %s at line %d, column %d", e.Found, strings.Join(e.Expected, ", "), e.Pos.Line, e.Pos.Column)
}
