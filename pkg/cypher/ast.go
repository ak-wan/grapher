package cypher

import (
	"bytes"
	"fmt"
	"strings"
)

// Query represents the Cypher query root element.
type Query struct {
	Root *SingleQuery
}

func (q Query) String() string {
	return q.Root.String()
}

// SingleQuery ...
type SingleQuery struct {
	Reading     []ReadingClause
	Distinct    bool
	ReturnItems []Expr
	Order       []OrderBy
	Skip        *Expr
	Limit       *Expr
}

func (sq SingleQuery) String() string {
	var buf bytes.Buffer

	for _, r := range sq.Reading {
		buf.WriteString(r.String())
	}

	buf.WriteString(" RETURN ")

	if sq.Distinct {
		buf.WriteString("DISTINCT ")
	}

	for _, i := range sq.ReturnItems {
		buf.WriteString(i.String())
	}

	if len(sq.Order) > 0 {
		buf.WriteString(" ORDER BY ")
		for _, o := range sq.Order {
			buf.WriteString(o.String())
		}
	}

	return buf.String()
}

// ReadingClause ...
type ReadingClause struct {
	OptionalMatch bool
	Pattern       []MatchPattern
	Where         *Expr
	// Unwind
	// Call
}

func (rc ReadingClause) String() string {
	var buf bytes.Buffer

	if rc.OptionalMatch {
		buf.WriteString(" OPTIONAL ")
	}

	for _, p := range rc.Pattern {
		buf.WriteString(p.String())
	}

	if w := rc.Where; w != nil {
		buf.WriteString((*w).String())
	}

	return buf.String()
}

// MatchPattern ...
type MatchPattern struct {
	Variable *Variable
	Elements []PatternElement
}

func (mp MatchPattern) String() string {
	var buf bytes.Buffer

	buf.WriteString("MATCH ")

	if mp.Variable != nil {
		buf.WriteString((*mp.Variable).String())
		buf.WriteString(" = ")
	}

	for _, e := range mp.Elements {
		buf.WriteString(e.String())
	}

	return buf.String()
}

// PatternElement ...
type PatternElement interface {
	patternElem()
	String() string
}

func (np NodePattern) patternElem() {}
func (ep EdgePattern) patternElem() {}

// NodePattern ...
type NodePattern struct {
	Variable   *Variable
	Labels     []string
	Properties map[string]Expr
}

func (np NodePattern) String() string {
	var buf bytes.Buffer

	buf.WriteRune('(')

	if np.Variable != nil {
		buf.WriteString((*np.Variable).String())
	}

	for _, l := range np.Labels {
		buf.WriteString(" :")
		buf.WriteString(l)
	}

	buf.WriteRune(')')

	return buf.String()
}

type EdgePattern struct {
	Direction  EdgeDirection   // 方向（->, <-）
	Variable   *string         // 关系变量（可选）
	RelTypes   []string        // 关系类型列表（如 ["KNOWS"]）
	Properties map[string]Expr // 属性键值对（可选）
	MinHops    *int            // 最小跳数（可变长度路径）
	MaxHops    *int            // 最大跳数（可变长度路径）
}

// Var ...
func (ep EdgePattern) Var() *string {
	return ep.Variable
}

func (ep EdgePattern) String() string {
	var buf bytes.Buffer

	switch ep.Direction {
	case EdgeRight, EdgeUndefined:
		buf.WriteRune('-')
	case EdgeLeft, EdgeOutgoing:
		buf.WriteString("<-")
	}

	buf.WriteRune('[')

	if ep.Variable != nil {
		buf.WriteString(*ep.Variable)
	}

	// 添加关系类型
	if len(ep.RelTypes) > 0 {
		buf.WriteString(":")
		buf.WriteString(strings.Join(ep.RelTypes, "|"))
	}

	if len(ep.Properties) > 0 {
		buf.WriteRune('{')

		var next bool
		for p, v := range ep.Properties {
			if next {
				buf.WriteRune(',')
			}
			buf.WriteString(p)
			buf.WriteRune(':')
			buf.WriteString(v.String())
			next = true
		}

		buf.WriteRune('}')
	}

	buf.WriteRune(']')

	switch ep.Direction {
	case EdgeLeft, EdgeUndefined:
		buf.WriteRune('-')
	case EdgeRight, EdgeOutgoing:
		buf.WriteString("->")
	}

	return buf.String()
}

// EdgeDirection ...
type EdgeDirection int

const (
	EdgeUndefined EdgeDirection = iota
	EdgeRight
	EdgeLeft
	EdgeOutgoing
)

// OrderDirection ...
type OrderDirection int

const (
	// Ascending defines the ascending ordering.
	Ascending OrderDirection = iota
	// Descending defines the descending ordering.
	Descending
)

// OrderBy ...
type OrderBy struct {
	Dir  OrderDirection
	Item Expr
}

func (o OrderBy) String() string {
	return ""
}

// Variable ...
type Variable string

func (v Variable) String() string {
	return string(v)
}

// Symbol ...
type Symbol string

func (s Symbol) String() string {
	return string(s)
}

// StrLiteral ...
type StrLiteral string

func (s StrLiteral) String() string {
	return fmt.Sprintf("\"%s\"", string(s))
}

// 在 cypher 包的 AST 类型定义部分（如 ast.go）添加
type IntegerLiteral int

func (i IntegerLiteral) exp() {} // 实现 Expr 接口
func (i IntegerLiteral) String() string { // 字符串表示
	return fmt.Sprintf("%d", i)
}

// Expr ...
type Expr interface {
	exp()
	String() string
}

func (v Variable) exp()   {}
func (s Symbol) exp()     {}
func (s StrLiteral) exp() {}
