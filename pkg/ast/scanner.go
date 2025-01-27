package ast

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// Scanner 表示词法扫描器
type Scanner struct {
	r *reader
}

// NewScanner 返回一个新的 Scanner 实例
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: &reader{r: bufio.NewReader(r)}}
}

// Scan 从输入中返回下一个 token
func (s *Scanner) Scan() (Token, Pos, string) {
	// 读取下一个码点
	ch0, pos := s.r.read()

	// 如果是空白字符，则消费所有连续空白
	// 如果是字母或特定可接受的特殊字符，则作为标识符或保留字处理
	if isWhitespace(ch0) {
		return s.scanWhitespace()
	} else if isLetter(ch0) || ch0 == '_' {
		s.r.unread()
		return s.scanIdent(true)
	} else if isDigit(ch0) {
		return s.scanNumber()
	}

	// 其他情况单独解析字符
	switch ch0 {
	case eof:
		return EOF, pos, tokens[EOF]
	case '"':
		return s.scanString()
	case '\'':
		return s.scanString()
	case '`':
		s.r.unread()
		return s.scanIdent(false)
	case '+':
		if ch1, _ := s.r.read(); ch1 == '=' {
			return INC, pos, tokens[INC]
		}
		s.r.unread()
		return PLUS, pos, tokens[PLUS]
	case '*':
		return MUL, pos, tokens[MUL]
	case '%':
		return MOD, pos, tokens[MOD]
	case '(':
		return LPAREN, pos, tokens[LPAREN]
	case ')':
		return RPAREN, pos, tokens[RPAREN]
	case '{':
		return LBRACE, pos, tokens[LBRACE]
	case '}':
		return RBRACE, pos, tokens[RBRACE]
	case '[':
		startPos := pos
		// 预读检查是否是关系范围语法 [*...]
		if ch1, _ := s.r.read(); ch1 == '*' {
			// 持续扫描直到闭合 ]
			return s.scanRelRange(startPos)
		}
		s.r.unread()
		return LBRACKET, pos, tokens[LBRACKET]
	case ']':
		return RBRACKET, pos, tokens[RBRACKET]
	case ',':
		return COMMA, pos, tokens[COMMA]
	case ';':
		return SEMICOLON, pos, tokens[SEMICOLON]
	case ':':
		return COLON, pos, tokens[COLON]
	case '-':
		startPos := pos
		ch1, _ := s.r.read()
		if ch1 == '>' {
			return EDGE_RIGHT, startPos, tokens[EDGE_RIGHT]
		}
		s.r.unread()
		return SUB, startPos, tokens[SUB]
	case '=':
		return EQ, pos, tokens[EQ]
	case '.':
		if ch1, _ := s.r.read(); ch1 == '.' {
			return DOUBLEDOT, pos, tokens[DOUBLEDOT]
		}
		s.r.unread()
		return DOT, pos, tokens[DOT]
	case '|':
		return BAR, pos, tokens[BAR]
	case '<':
		ch1, _ := s.r.read()
		if ch1 == '>' {
			return NEQ, pos, tokens[NEQ]
		} else if ch1 == '=' {
			return LTE, pos, tokens[LTE]
		} else if ch1 == '-' {
			return EDGE_LEFT, pos, tokens[EDGE_LEFT]
		}
		s.r.unread()
		return LT, pos, tokens[LT]
	case '>':
		if ch1, _ := s.r.read(); ch1 == '=' {
			return GTE, pos, tokens[GTE]
		}
		s.r.unread()
		return GT, pos, tokens[GT]
	case '/':
		ch1, _ := s.r.read()
		if ch1 == '*' {
			if err := s.skipUntilEndComment(); err != nil {
				return ILLEGAL, pos, "/*"
			}
			return COMMENT, pos, "/*"
		} else if ch1 == '/' {
			s.skipUntilNewline()
			return COMMENT, pos, "//"
		}
		s.r.unread()
		return DIV, pos, tokens[DIV]
	}

	return ILLEGAL, pos, string(ch0)
}

// scanWhitespace 消费当前及后续所有连续空白字符
func (s *Scanner) scanWhitespace() (tok Token, pos Pos, lit string) {
	var buf bytes.Buffer
	ch, pos := s.r.curr()
	_, _ = buf.WriteRune(ch)

	for {
		ch, _ = s.r.read()
		if ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.r.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	return WS, pos, buf.String()
}

// scanIdent 扫描标识符（可选择是否查找保留字）
func (s *Scanner) scanIdent(lookup bool) (tok Token, pos Pos, lit string) {
	_, pos = s.r.read()
	s.r.unread()

	var buf bytes.Buffer
	for {
		if ch, _ := s.r.read(); ch == eof {
			break
		} else if ch == '"' || ch == '\'' || ch == '`' {
			tok0, pos0, lit0 := s.scanString()
			if tok0 == BADSTRING || tok0 == BADESCAPE {
				return tok0, pos0, lit0
			}
			return IDENT, pos, lit0
		} else if isIdentChar(ch) {
			s.r.unread()
			buf.WriteString(ScanBareIdent(s.r))
		} else {
			s.r.unread()
			break
		}
	}
	lit = buf.String()

	// 如果匹配关键字则返回对应 token
	if lookup {
		if tok = Lookup(lit); tok != IDENT {
			return tok, pos, lit
		}
	}
	return IDENT, pos, lit
}

// scanString 扫描带引号的字符串（支持转义）
func (s *Scanner) scanString() (tok Token, pos Pos, lit string) {
	s.r.unread()
	_, pos = s.r.curr()

	var err error
	lit, err = ScanString(s.r)
	if err == errBadString {
		return BADSTRING, pos, lit
	} else if err == errBadEscape {
		_, pos = s.r.curr()
		return BADESCAPE, pos, lit
	}
	return STRING, pos, lit
}

// ScanString 从符文读取器中读取带引号的字符串
func ScanString(r io.RuneScanner) (string, error) {
	ending, _, err := r.ReadRune()
	if err != nil {
		return "", errBadString
	}

	var buf bytes.Buffer
	for {
		ch0, _, err := r.ReadRune()
		if ch0 == ending {
			return buf.String(), nil
		} else if err != nil || ch0 == '\n' {
			return buf.String(), errBadString
		} else if ch0 == '\\' {
			// 处理转义字符
			ch1, _, _ := r.ReadRune()
			if ch1 == 'n' {
				_, _ = buf.WriteRune('\n')
			} else if ch1 == '\\' {
				_, _ = buf.WriteRune('\\')
			} else if ch1 == '"' {
				_, _ = buf.WriteRune('"')
			} else if ch1 == '\'' {
				_, _ = buf.WriteRune('\'')
			} else if ch1 == '`' {
				_, _ = buf.WriteRune('`')
			} else {
				return string(ch0) + string(ch1), errBadEscape
			}
		} else {
			_, _ = buf.WriteRune(ch0)
		}
	}
}

var errBadString = errors.New("bad string")
var errBadEscape = errors.New("bad escape")

// ScanBareIdent 读取裸标识符
func ScanBareIdent(r io.RuneScanner) string {
	var buf bytes.Buffer
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			break
		} else if !isIdentChar(ch) {
			r.UnreadRune()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return buf.String()
}

// isWhitespace 判断符文是否为空白字符（空格/制表符/换行）
func isWhitespace(ch rune) bool { return ch == ' ' || ch == '\t' || ch == '\n' }

// isLetter 判断是否为字母
func isLetter(ch rune) bool { return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') }

// isDigit 判断是否为数字
func isDigit(ch rune) bool { return (ch >= '0' && ch <= '9') }

// isIdentChar 判断是否可作为标识符字符
func isIdentChar(ch rune) bool { return isLetter(ch) || isDigit(ch) || ch == '_' }

// scanNumber 扫描数字字面量
func (s *Scanner) scanNumber() (tok Token, pos Pos, lit string) {
	var buf bytes.Buffer

	ch, pos := s.r.curr()
	if ch == '.' {
		// 检查后续是否为数字
		ch1, _ := s.r.read()
		s.r.unread()
		if !isDigit(ch1) {
			return ILLEGAL, pos, "."
		}
		s.r.unread()
	} else {
		s.r.unread()
	}

	// 读取整数部分
	_, _ = buf.WriteString(s.scanDigits())

	// 处理小数部分
	isDecimal := false
	if ch0, _ := s.r.read(); ch0 == '.' {
		isDecimal = true
		if ch1, _ := s.r.read(); isDigit(ch1) {
			_, _ = buf.WriteRune(ch0)
			_, _ = buf.WriteRune(ch1)
			_, _ = buf.WriteString(s.scanDigits())
		} else {
			s.r.unread()
		}
	} else {
		s.r.unread()
	}

	if !isDecimal {
		s.r.unread()
		return INTEGER, pos, buf.String()
	}
	return NUMBER, pos, buf.String()
}

// scanDigits 扫描连续数字
func (s *Scanner) scanDigits() string {
	var buf bytes.Buffer
	for {
		ch, _ := s.r.read()
		if !isDigit(ch) {
			s.r.unread()
			break
		}
		_, _ = buf.WriteRune(ch)
	}
	return buf.String()
}

// skipUntilNewline 跳过直到换行符
func (s *Scanner) skipUntilNewline() {
	for {
		if ch, _ := s.r.read(); ch == '\n' || ch == eof {
			return
		}
	}
}

// skipUntilEndComment 跳过直到注释结束符 */
func (s *Scanner) skipUntilEndComment() error {
	for {
		if ch1, _ := s.r.read(); ch1 == '*' {
		star:
			ch2, _ := s.r.read()
			if ch2 == '/' {
				return nil
			} else if ch2 == '*' {
				goto star
			} else if ch2 == eof {
				return io.EOF
			}
		} else if ch1 == eof {
			return io.EOF
		}
	}
}

// bufScanner 带缓冲区的扫描器包装
type bufScanner struct {
	s   *Scanner
	i   int // 缓冲区索引
	n   int // 缓冲数量
	buf [3]struct {
		tok Token
		pos Pos
		lit string
	}
}

// newBufScanner 创建带缓冲的扫描器
func newBufScanner(r io.Reader) *bufScanner {
	return &bufScanner{s: NewScanner(r)}
}

// Scan 读取下一个 token
func (s *bufScanner) Scan() (tok Token, pos Pos, lit string) {
	if s.n > 0 {
		s.n--
		return s.curr()
	}

	s.i = (s.i + 1) % len(s.buf)
	buf := &s.buf[s.i]
	buf.tok, buf.pos, buf.lit = s.s.Scan()
	fmt.Printf("Tokens:%s\t\tContent: \"%s\"\n", buf.tok, buf.lit)
	return s.curr()
}

// scanRelRange 扫描关系范围语法 [*...]
func (s *Scanner) scanRelRange(startPos Pos) (Token, Pos, string) {
	var buf bytes.Buffer
	buf.WriteRune('[')
	buf.WriteRune('*')

	for {
		ch, _ := s.r.read()
		if ch == eof {
			return ILLEGAL, startPos, buf.String()
		}
		buf.WriteRune(ch)
		if ch == ']' {
			return REL_RANGE, startPos, buf.String()
		}
	}
}

// Unscan 回退上一个 token
func (s *bufScanner) Unscan() { s.n++ }

// curr 获取当前 token
func (s *bufScanner) curr() (tok Token, pos Pos, lit string) {
	buf := &s.buf[(s.i-s.n+len(s.buf))%len(s.buf)]
	return buf.tok, buf.pos, buf.lit
}

// reader 带缓冲的符文读取器
type reader struct {
	r   io.RuneScanner
	i   int // 缓冲区索引
	n   int // 未读计数
	pos Pos // 全局位置
	buf [3]struct {
		ch  rune
		pos Pos
	}
	eof bool
}

// ReadRune 实现 io.RuneReader 接口
func (r *reader) ReadRune() (ch rune, size int, err error) {
	ch, _ = r.read()
	if ch == eof {
		err = io.EOF
	}
	return
}

// UnreadRune 实现 io.RuneScanner 接口
func (r *reader) UnreadRune() error {
	r.unread()
	return nil
}

// read 读取下一个符文及其位置
func (r *reader) read() (ch rune, pos Pos) {
	if r.n > 0 {
		r.n--
		buf := &r.buf[(r.i-r.n+len(r.buf))%len(r.buf)]
		return buf.ch, buf.pos
	}

	ch, _, err := r.r.ReadRune()
	if err != nil {
		ch = eof
	}

	// 处理 Windows 换行符 \r\n
	if ch == '\r' {
		if nextCh, _, err := r.r.ReadRune(); err == nil && nextCh == '\n' {
			ch = '\n'
		} else if err == nil {
			_ = r.r.UnreadRune()
		}
	}

	pos = r.pos

	if ch == '\n' {
		r.pos.Line++
		r.pos.Column = 1
		r.pos.Offset++
	} else if ch != eof {
		r.pos.Column++
		r.pos.Offset++
	}

	r.i = (r.i + 1) % len(r.buf)
	buf := &r.buf[r.i]
	buf.ch = ch
	buf.pos = pos

	return ch, pos
}

// unread 回退到上一个字符
func (r *reader) unread() {
	if r.n >= len(r.buf) {
		panic("缓冲区溢出")
	}

	r.n++

	if r.n > 0 {
		idx := (r.i - r.n + len(r.buf)) % len(r.buf)
		prevPos := r.buf[idx].pos
		r.pos = prevPos
	}
}

// curr 获取当前字符
func (r *reader) curr() (ch rune, pos Pos) {
	i := (r.i - r.n + len(r.buf)) % len(r.buf)
	buf := &r.buf[i]
	return buf.ch, buf.pos
}

const eof = rune(0)