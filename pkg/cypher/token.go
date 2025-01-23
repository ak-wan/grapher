package cypher

import "strings"

// Token 是 Cypher 语言的词法标记
type Token int

const (
	// ILLEGAL Token, EOF, WS 特殊的tokens
	ILLEGAL Token = iota
	EOF
	WS
	COMMENT

	literalBeg
	// 字面量标记
	IDENT     // main
	NUMBER    // 12345.67
	INTEGER   // 12345
	STRING    // "abc"
	BADSTRING // "abc
	BADESCAPE // "\q
	TRUE      // true
	FALSE     // false
	NULL      // null
	literalEnd

	// 操作符标记
	operatorBeg
	PLUS // +
	SUB  // -
	MUL  // *
	DIV  // /
	MOD  // %
	POW  // ^
	EQ   // =
	NEQ  // <>
	LT   // <
	LTE  // <=
	GT   // >
	GTE  // >=
	INC  // +=
	BAR  // |

	AND // AND
	OR  // OR
	XOR // XOR
	NOT // NOT
	operatorEnd

	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
	COMMA     // ,
	COLON     // :
	SEMICOLON // ;
	DOT       // .
	DOUBLEDOT // ..

	keywordBeg
	// 关键字标记
	ADD
	ALL
	AS
	ASC
	ASCENDING
	BY
	CASE
	CONSTRAINT
	CONTAINS
	CREATE
	DELETE
	DESC
	DESCENDING
	DETACH
	DISTINCT
	DO
	DROP
	ELSE
	END
	ENDS
	EXISTS
	FOR
	IN
	IS
	LIMIT
	MANDATORY
	MATCH
	MERGE
	OF
	ON
	OPTIONAL
	ORDER
	REMOVE
	REQUIRE
	RETURN
	SCALAR
	SET
	SKIP
	STARTS
	THEN
	UNION
	UNIQUE
	UNWIND
	WHEN
	WHERE
	WITH
	keywordEnd
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	WS:      "WS",

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

	LPAREN:    "(",
	RPAREN:    ")",
	LBRACE:    "{",
	RBRACE:    "}",
	LBRACKET:  "[",
	RBRACKET:  "]",
	COMMA:     ",",
	COLON:     ":",
	SEMICOLON: ";",
	DOT:       ".",

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

var keywords map[string]Token

func init() {
	keywords = make(map[string]Token)
	for t := keywordBeg + 1; t < keywordEnd; t++ {
		keywords[strings.ToLower(tokens[t])] = t
	}
	for _, t := range []Token{AND, OR, XOR, NOT} {
		keywords[strings.ToLower(tokens[t])] = t
	}
	keywords["true"] = TRUE
	keywords["false"] = FALSE
	keywords["null"] = NULL
}

// isOperator 对于操作符标记返回 true。
func (t Token) IsOperator() bool { return t > operatorBeg && t < operatorEnd }

// String 返回标记的字符串表示形式。
func (t Token) String() string {
	if t >= 0 && t < Token(len(tokens)) {
		return tokens[t]
	}
	return ""
}

func tokstr(tok Token, lit string) string {
	if lit != "" {
		return lit
	}
	return tok.String()
}

// Lookup 返回标识符的标记
func Lookup(ident string) Token {
	if t, ok := keywords[strings.ToLower(ident)]; ok {
		return t
	}
	return IDENT
}

// Pos specifies the line and character position of a token.
// The Char and Line are both zero-based indexes.
type Pos struct {
	Line int
	Char int
}

