package ast

import (
	"bytes"
	"fmt"
	"strings"
)

// SingleQuery 表示单个查询语句（如 MATCH-RETURN 结构）
type SingleQuery struct {
	Reading     []ReadingClause // 读取子句（MATCH/OPTIONAL MATCH）
	Distinct    bool            // 是否去重
	ReturnItems []Expr          // RETURN 返回项
	Order       []OrderBy       // 排序规则
	Skip        *Expr           // 跳过行数
	Limit       *Expr           // 限制行数
}

func (sq SingleQuery) String() string {
	var buf bytes.Buffer

	// 拼接所有 READING 子句
	for _, r := range sq.Reading {
		buf.WriteString(r.String())
	}

	buf.WriteString(" RETURN ")

	// 处理 DISTINCT
	if sq.Distinct {
		buf.WriteString("DISTINCT ")
	}

	// 拼接返回项
	for _, i := range sq.ReturnItems {
		buf.WriteString(i.String())
	}

	// 处理排序子句
	if len(sq.Order) > 0 {
		buf.WriteString(" ORDER BY ")
		for _, o := range sq.Order {
			buf.WriteString(o.String())
		}
	}

	return buf.String()
}

// ReadingClause 表示查询中的读取子句（MATCH/UNWIND/CALL 等）
type ReadingClause struct {
	OptionalMatch bool           // 是否是 OPTIONAL MATCH
	Pattern       []MatchPattern // 匹配模式
	Where         *Expr          // WHERE 条件
}

func (rc ReadingClause) String() string {
	var buf bytes.Buffer

	// 处理 OPTIONAL MATCH
	if rc.OptionalMatch {
		buf.WriteString(" OPTIONAL ")
	}

	// 拼接匹配模式
	for _, p := range rc.Pattern {
		buf.WriteString(p.String())
	}

	// 处理 WHERE 条件
	if w := rc.Where; w != nil {
		buf.WriteString((*w).String())
	}

	return buf.String()
}

// MatchPattern 表示 MATCH 子句中的模式
type MatchPattern struct {
	Variable *Variable        // 模式变量（可选）
	Elements []PatternElement // 模式元素（节点/边）
}

func (mp MatchPattern) String() string {
	var buf bytes.Buffer

	buf.WriteString("MATCH ")

	// 处理模式变量赋值（如 path = (a)-[...]->(b)）
	if mp.Variable != nil {
		buf.WriteString((*mp.Variable).String())
		buf.WriteString(" = ")
	}

	// 拼接模式元素
	for _, e := range mp.Elements {
		buf.WriteString(e.String())
	}

	return buf.String()
}

// PatternElement 模式元素接口（节点或边）
type PatternElement interface {
	patternElem()
	String() string
}

func (np NodePattern) patternElem() {}
func (ep EdgePattern) patternElem() {}

// NodePattern 表示节点模式（如 (a:Person {name: 'Alice'}）)
type NodePattern struct {
	Variable   *Variable       // 节点变量
	Labels     []string        // 节点标签列表
	Properties map[string]Expr // 节点属性
}

func (np NodePattern) String() string {
	var buf bytes.Buffer

	buf.WriteRune('(')

	// 处理变量名
	if np.Variable != nil {
		buf.WriteString((*np.Variable).String())
	}

	// 处理标签
	for _, l := range np.Labels {
		buf.WriteString(" :")
		buf.WriteString(l)
	}

	buf.WriteRune(')')

	return buf.String()
}

// EdgePattern 表示边模式（如 -[r:KNOWS {since: 2010}]->）
type EdgePattern struct {
	Direction  EdgeDirection   // 方向（->, <-）
	Variable   *string         // 关系变量（可选）
	RelTypes   []string        // 关系类型列表（如 ["KNOWS"]）
	Properties map[string]Expr // 属性键值对（可选）
	MinHops    *int            // 最小跳数（可变长度路径）
	MaxHops    *int            // 最大跳数（可变长度路径）
}

// Var 返回关系变量（可选）
func (ep EdgePattern) Var() *string {
	return ep.Variable
}

func (ep EdgePattern) String() string {
	var buf bytes.Buffer

	// 处理左边方向
	switch ep.Direction {
	case EdgeRight, EdgeUndefined:
		buf.WriteRune('-')
	case EdgeLeft, EdgeOutgoing:
		buf.WriteString("<-")
	}

	buf.WriteRune('[')

	// 处理关系变量
	if ep.Variable != nil {
		buf.WriteString(*ep.Variable)
	}

	// 添加关系类型
	if len(ep.RelTypes) > 0 {
		buf.WriteString(":")
		buf.WriteString(strings.Join(ep.RelTypes, "|"))
	}

	// 处理属性
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

	// 处理右边方向
	switch ep.Direction {
	case EdgeLeft, EdgeUndefined:
		buf.WriteRune('-')
	case EdgeRight, EdgeOutgoing:
		buf.WriteString("->")
	}

	return buf.String()
}

// EdgeDirection 边方向枚举
type EdgeDirection int

const (
	EdgeUndefined EdgeDirection = iota // 未定义方向
	EdgeRight                          // 右方向 ->
	EdgeLeft                           // 左方向 <-
	EdgeOutgoing                       // 出方向（兼容处理）
)

// OrderDirection 排序方向枚举
type OrderDirection int

const (
	Ascending  OrderDirection = iota // 升序排列
	Descending                       // 降序排列
)

// OrderBy 排序规则定义
type OrderBy struct {
	Dir  OrderDirection // 排序方向
	Item Expr           // 排序表达式
}

func (o OrderBy) String() string {
	return ""
}

// Variable 表示变量（如 MATCH (a) 中的 a）
type Variable string

func (v Variable) String() string {
	return string(v)
}

// Symbol 表示符号（保留字或特殊符号）
type Symbol string

func (s Symbol) String() string {
	return string(s)
}

// StrLiteral 表示字符串字面量
type StrLiteral string

func (s StrLiteral) String() string {
	return fmt.Sprintf("\"%s\"", string(s))
}

// IntegerLiteral 表示整数字面量
type IntegerLiteral int

func (i IntegerLiteral) exp() {} // 实现 Expr 接口标记方法
func (i IntegerLiteral) String() string {
	return fmt.Sprintf("%d", i)
}

// Expr 表示 Cypher 中的表达式接口
type Expr interface {
	exp()
	String() string
}

// 实现 Expr 接口
func (v Variable) exp()   {}
func (s Symbol) exp()     {}
func (s StrLiteral) exp() {}
