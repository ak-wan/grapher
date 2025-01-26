package cypher

import "strings"

// Token 表示 Cypher 语言的词法单元类型
type Token int

const (
	// 特殊标记
	ILLEGAL   Token = iota // 非法字符
	EOF                    // 文件结束符
	WS                     // 空白字符
	COMMENT                // 注释
	REL_RANGE              // 关系范围语法 [*...]

	literalBeg // 字面量标记开始
	IDENT      // 标识符（如变量名）
	NUMBER     // 浮点数字面量（如 12345.67）
	INTEGER    // 整数字面量（如 12345）
	STRING     // 字符串字面量（如 "abc"）
	BADSTRING  // 不完整字符串（如 "abc）
	BADESCAPE  // 错误转义字符（如 \q）
	TRUE       // 布尔值 true
	FALSE      // 布尔值 false
	NULL       // 空值 null
	literalEnd // 字面量标记结束

	operatorBeg // 操作符标记开始
	// 基础算术运算符
	PLUS // +
	SUB  // -
	MUL  // *
	DIV  // /
	MOD  // %
	POW  // ^

	// 逻辑运算符
	AND // AND
	OR  // OR
	XOR // XOR
	NOT // NOT

	// 比较运算符
	EQ  // =
	NEQ // <>
	LT  // <
	LTE // <=
	GT  // >
	GTE // >=

	// 复合运算符
	INC         // +=
	BAR         // |
	operatorEnd // 操作符标记结束

	// 标点符号
	LPAREN     // (
	RPAREN     // )
	LBRACE     // {
	RBRACE     // }
	LBRACKET   // [
	RBRACKET   // ]
	COMMA      // ,
	COLON      // :
	SEMICOLON  // ;
	DOT        // .
	DOUBLEDOT  // ..
	EDGE_RIGHT // ->
	EDGE_LEFT  // <-

	keywordBeg // 关键字标记开始
	ADD        // ADD
	ALL        // ALL
	AS         // AS
	ASC        // ASC
	ASCENDING  // ASCENDING
	BY         // BY
	CASE       // CASE
	CONSTRAINT // CONSTRAINT
	CONTAINS   // CONTAINS
	CREATE     // CREATE
	DELETE     // DELETE
	DESC       // DESC
	DESCENDING // DESCENDING
	DETACH     // DETACH
	DISTINCT   // DISTINCT
	DO         // DO
	DROP       // DROP
	ELSE       // ELSE
	END        // END
	ENDS       // ENDS
	EXISTS     // EXISTS
	FOR        // FOR
	IN         // IN
	IS         // IS
	LIMIT      // LIMIT
	MANDATORY  // MANDATORY
	MATCH      // MATCH
	MERGE      // MERGE
	OF         // OF
	ON         // ON
	OPTIONAL   // OPTIONAL
	ORDER      // ORDER
	REMOVE     // REMOVE
	REQUIRE    // REQUIRE
	RETURN     // RETURN
	SCALAR     // SCALAR（标量）
	SET        // SET
	SKIP       // SKIP
	STARTS     // STARTS
	THEN       // THEN
	UNION      // UNION
	UNIQUE     // UNIQUE
	UNWIND     // UNWIND
	WHEN       // WHEN
	WHERE      // WHERE
	WITH       // WITH
	keywordEnd // 关键字标记结束
)

// tokens 定义每个 Token 的字符串表示
var tokens = [...]string{
	ILLEGAL:   "ILLEGAL",
	EOF:       "EOF",
	WS:        "WS",
	REL_RANGE: "REL_RANGE",

	IDENT:  "IDENT",
	NUMBER: "NUMBER",
	STRING: "STRING",
	TRUE:   "TRUE",
	FALSE:  "FALSE",

	PLUS: "+",
	SUB:  "-",
	MUL:  "*",
	DIV:  "/",
	MOD:  "%",
	POW:  "^",

	AND: "AND",
	OR:  "OR",
	XOR: "XOR",
	NOT: "NOT",

	EQ:  "=",
	NEQ: "<>",
	LT:  "<",
	LTE: "<=",
	GT:  ">",
	GTE: ">=",

	LPAREN:     "(",
	RPAREN:     ")",
	LBRACE:     "{",
	RBRACE:     "}",
	LBRACKET:   "[",
	RBRACKET:   "]",
	COMMA:      ",",
	COLON:      ":",
	SEMICOLON:  ";",
	DOT:        ".",
	DOUBLEDOT:  "..",
	EDGE_RIGHT: "->",
	EDGE_LEFT:  "<-",

	ADD:        "ADD",
	ALL:        "ALL",
	AS:         "AS",
	ASC:        "ASC",
	ASCENDING:  "ASCENDING",
	BY:         "BY",
	CASE:       "CASE",
	CONSTRAINT: "CONSTRAINT",
	CONTAINS:   "CONTAINS",
	CREATE:     "CREATE",
	DELETE:     "DELETE",
	DESC:       "DESC",
	DESCENDING: "DESCENDING",
	DETACH:     "DETACH",
	DISTINCT:   "DISTINCT",
	DO:         "DO",
	DROP:       "DROP",
	ELSE:       "ELSE",
	END:        "END",
	ENDS:       "ENDS",
	EXISTS:     "EXISTS",
	FOR:        "FOR",
	IN:         "IN",
	IS:         "IS",
	LIMIT:      "LIMIT",
	MANDATORY:  "MANDATORY",
	MATCH:      "MATCH",
	MERGE:      "MERGE",
	OF:         "OF",
	ON:         "ON",
	OPTIONAL:   "OPTIONAL",
	ORDER:      "ORDER",
	REMOVE:     "REMOVE",
	REQUIRE:    "REQUIRE",
	RETURN:     "RETURN",
	SCALAR:     "SCALAR",
	SET:        "SET",
	SKIP:       "SKIP",
	STARTS:     "STARTS",
	THEN:       "THEN",
	UNION:      "UNION",
	UNIQUE:     "UNIQUE",
	UNWIND:     "UNWIND",
	WHEN:       "WHEN",
	WHERE:      "WHERE",
	WITH:       "WITH",
}

// 关键字映射表（小写 -> Token）
var keywords map[string]Token

func init() {
	keywords = make(map[string]Token)
	// 注册所有关键字
	for t := keywordBeg + 1; t < keywordEnd; t++ {
		keywords[strings.ToLower(tokens[t])] = t
	}
	// 额外注册逻辑运算符
	for _, t := range []Token{AND, OR, XOR, NOT} {
		keywords[strings.ToLower(tokens[t])] = t
	}
	// 注册布尔值和空值
	keywords["true"] = TRUE
	keywords["false"] = FALSE
	keywords["null"] = NULL
}

// IsOperator 判断是否为操作符类型的 Token
func (t Token) IsOperator() bool { return t > operatorBeg && t < operatorEnd }

// String 返回 Token 的可读字符串表示
func (t Token) String() string {
	if t >= 0 && t < Token(len(tokens)) {
		return tokens[t]
	}
	return ""
}

// tokstr 辅助函数：优先返回字面量值
func tokstr(tok Token, lit string) string {
	if lit != "" {
		return lit
	}
	return tok.String()
}

// Lookup 通过标识符查找对应的 Token（支持关键字）
func Lookup(ident string) Token {
	if t, ok := keywords[strings.ToLower(ident)]; ok {
		return t
	}
	return IDENT
}

// Pos 表示源码中的位置信息
type Pos struct {
	Line   int // 行号（从1开始）
	Column int // 列号（从1开始）
	Offset int // 字节偏移量（从0开始）
}
