package cypher

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Parser represents a Cypher parser.
type Parser struct {
	s *bufScanner
}

// NewParser returns a new instance of Parser.
func NewParser(r io.Reader) *Parser {
	return &Parser{s: newBufScanner(r)}
}

// ParseQuery parses a query string and returns its AST representation.
func ParseQuery(s string) (Query, error) {
	return NewParser(strings.NewReader(s)).ParseQuery()
}

// ParseQuery parses a Cypher string and returns a Query AST object.
func (p *Parser) ParseQuery() (q Query, err error) {
	for {
		if tok, _, _ := p.ScanIgnoreWhitespace(); tok == EOF {
			return q, nil
		} else if tok == SEMICOLON {
			continue
		} else {
			p.Unscan()
			q.Root, err = p.ParseSingleQuery()
			if err != nil {
				return q, err
			}
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

// ScanReadingClause ...
func (p *Parser) ScanReadingClause() (*ReadingClause, error) {
	rc := &ReadingClause{}

	// might be optionally matching this
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok == OPTIONAL {
		rc.OptionalMatch = true
	} else {
		p.Unscan()
	}

	// MATCH is obligatory here
	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != MATCH {
		return nil, newParseError(tokstr(tok, lit), []string{"MATCH"}, pos)
	}

	for {
		mp, err := p.ScanMatchPattern()
		if err != nil {
			return nil, err
		}
		rc.Pattern = append(rc.Pattern, *mp)

		if tok, _, _ := p.ScanIgnoreWhitespace(); tok != COMMA {
			p.Unscan()
			break
		}
	}

	// might be optional WHERE
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

// ScanMatchPattern ...
func (p *Parser) ScanMatchPattern() (*MatchPattern, error) {
	mp := &MatchPattern{}

	if tok, _, lit := p.ScanIgnoreWhitespace(); tok == IDENT {
		// We need the `=` character here
		if tok1, pos, lit1 := p.ScanIgnoreWhitespace(); tok1 != EQ {
			return nil, newParseError(tokstr(tok1, lit1), []string{"="}, pos)
		}

		v := Variable(lit)
		mp.Variable = &v
	} else {
		p.Unscan()
	}

	// scan the pattern itself
	elems, err := p.ScanPatternElements()
	if err != nil {
		return nil, err
	}
	mp.Elements = elems

	return mp, nil
}

// ScanPatternElements ...
func (p *Parser) ScanPatternElements() (pe []PatternElement, err error) {
	var node *NodePattern
	numParens := 0
	for {
		node, err = p.ScanNodePattern()
		if err != nil {
			return nil, err
		}
		if node == nil {
			// might be only parens around the actual match, lets try...
			if tok, pos, lit := p.ScanIgnoreWhitespace(); tok == LPAREN {
				numParens++
			} else {
				return nil, newParseError(tokstr(tok, lit), []string{"("}, pos)
			}
		} else {
			break
		}
	}

	pe = []PatternElement{node}

	for {
		edge, err := p.ScanEdgePattern()
		if err != nil {
			return nil, err
		} else if edge == nil {
			break
		} else {
			pe = append(pe, edge)
		}
		node, err = p.ScanNodePattern()
		if err != nil {
			return nil, err
		} else if node == nil {
			break
		} else {
			pe = append(pe, node)
		}
	}

	// need to close all open parens
	for i := 0; i < numParens; i++ {
		if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != RPAREN {
			return nil, newParseError(tokstr(tok, lit), []string{")"}, pos)
		}
	}

	return pe, nil
}

// ScanNodePattern returns a NodePattern if possible to consume a complete valid node.
func (p *Parser) ScanNodePattern() (*NodePattern, error) {
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok != LPAREN {
		// We already know we cannot consume a valid node if the pattern doesn't start with `(`
		p.Unscan()
		return nil, nil
	}
	var validNode bool
	var node NodePattern
	if tok, _, lit := p.ScanIgnoreWhitespace(); tok == IDENT {
		v := Variable(lit)
		node.Variable = &v
		validNode = true
	} else {
		p.Unscan()
	}

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

	props, err := p.ScanProperties()
	if err != nil {
		return nil, err
	} else if props != nil {
		node.Properties = *props
		validNode = true
	}

	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok == RPAREN {
		return &node, nil
	} else if validNode && tok != RPAREN {
		// We need to close the node definition
		return nil, newParseError(tokstr(tok, lit), []string{")"}, pos)
	}

	p.Unscan()
	p.Unscan() // unscan the first LPAREN then
	return nil, nil
}

func (p *Parser) ScanEdgePattern() (*EdgePattern, error) {
	ep := &EdgePattern{
		Direction: EdgeUndefined,
	}

	// 扫描边的起始方向
	tok1, _, _ := p.ScanIgnoreWhitespace()
	if tok1 == SUB {
		tok2, _, _ := p.ScanIgnoreWhitespace()
		if tok2 == GT {
			ep.Direction = EdgeRight
		} else {
			p.Unscan()
			ep.Direction = EdgeUndefined
		}
	} else if tok1 == LT {
		tok2, _, _ := p.ScanIgnoreWhitespace()
		if tok2 == SUB {
			ep.Direction = EdgeLeft
		} else {
			p.Unscan()
			return nil, newParseError("", []string{"-"}, Pos{})
		}
	} else {
		p.Unscan()
		return nil, nil
	}

	// 检查是否有方括号
	if tok, _, _ := p.ScanIgnoreWhitespace(); tok != LBRACKET {
		p.Unscan()
		return ep, nil
	}

	// 解析方括号内的内容
	for {
		tok, pos, lit := p.ScanIgnoreWhitespace()
		switch tok {
		case MUL:
			// 处理跳数范围
			minHops := 0
			ep.MinHops = &minHops
			// 检查是否有更多跳数定义
			nextTok, _, _ := p.ScanIgnoreWhitespace()
			if nextTok == DOUBLEDOT {
				maxTok, _, maxLit := p.ScanIgnoreWhitespace()
				if maxTok == INTEGER {
					max, _ := strconv.Atoi(maxLit)
					ep.MaxHops = &max
				}
			} else {
				p.Unscan()
			}
		case RBRACKET:
			// 结束方括号
			goto endEdge
		case IDENT:
			// 变量名
			v := lit
			ep.Variable = &v
		case COLON:
			// 标签
			labelTok, _, labelLit := p.ScanIgnoreWhitespace()
			if labelTok == IDENT {
				ep.Labels = append(ep.Labels, labelLit)
			} else {
				return nil, newParseError(tokstr(labelTok, labelLit), []string{"label"}, pos)
			}
		default:
			// 其他情况（如属性）
			p.Unscan()
			props, err := p.ScanProperties()
			if err != nil {
				return nil, err
			}
			if props != nil {
				ep.Properties = *props
			}
		}
	}

endEdge:
	// 处理边的结束方向
	tok3, _, _ := p.ScanIgnoreWhitespace()
	tok4, _, _ := p.ScanIgnoreWhitespace()
	if (tok3 == SUB && tok4 == GT) && ep.Direction == EdgeUndefined {
		ep.Direction = EdgeRight
	} else if (tok3 == LT && tok4 == SUB) && ep.Direction == EdgeUndefined {
		ep.Direction = EdgeLeft
	} else {
		p.Unscan()
		p.Unscan()
	}

	return ep, nil
}

// 基础表达式解析（需扩展支持更多类型）
func (p *Parser) ScanExpression() (Expr, error) {
	// 当前简化实现，需根据实际情况扩展
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

	if tok, pos, lit := p.ScanIgnoreWhitespace(); tok != RBRACE {
		return nil, newParseError(tokstr(tok, lit), []string{"}"}, pos)
	}

	return &props, nil
}

// Scan returns the next token from the underlying scanner.
func (p *Parser) Scan() (tok Token, pos Pos, lit string) { return p.s.Scan() }

// ScanIgnoreWhitespace scans the next non-whitespace and non-comment token.
func (p *Parser) ScanIgnoreWhitespace() (tok Token, pos Pos, lit string) {
	for {
		tok, pos, lit = p.Scan()
		if tok == WS || tok == COMMENT {
			continue
		}
		return
	}
}

// Unscan pushes the previously read token back onto the buffer.
func (p *Parser) Unscan() { p.s.Unscan() }

// ParseError represents an error that occurred during parsing.
type ParseError struct {
	Message  string
	Found    string
	Expected []string
	Pos      Pos
}

// newParseError returns a new instance of ParseError.
func newParseError(found string, expected []string, pos Pos) *ParseError {
	return &ParseError{Found: found, Expected: expected, Pos: pos}
}

// Error returns the string representation of the error.
func (e *ParseError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s at line %d, char %d", e.Message, e.Pos.Line+1, e.Pos.Char+1)
	}
	return fmt.Sprintf("found %s, expected %s at line %d, char %d", e.Found, strings.Join(e.Expected, ", "), e.Pos.Line+1, e.Pos.Char+1)
}
