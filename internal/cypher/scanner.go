package cypher

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// Scanner is a lexical scanner.
type Scanner struct {
	r *reader
}

// NewScanner returns a new instance of Scanner.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: &reader{r: bufio.NewReader(r)}}
}

// Scan returns the next token from the input.
func (s *Scanner) Scan() (Token, Pos, string) {
	// Read next code point.
	ch0, pos := s.r.read()

	// If we see whitespace then consume all contiguous whitespace.
	// If we see a letter, or certain acceptable special characters, then consume
	// as an ident or reserved word.
	if isWhitespace(ch0) {
		return s.scanWhitespace()
	} else if isLetter(ch0) || ch0 == '_' {
		s.r.unread()
		return s.scanIdent(true)
	} else if isDigit(ch0) {
		return s.scanNumber()
	}

	// Otherwise parse individual characters.
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
		// 预读下一个字符检查是否是 * 开头的关系范围
		if ch1, _ := s.r.read(); ch1 == '*' {
			// 继续扫描直到 ]，例如 [*1..5]
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

// scanWhitespace consumes the current rune and all contiguous whitespace.
func (s *Scanner) scanWhitespace() (tok Token, pos Pos, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	ch, pos := s.r.curr()
	_, _ = buf.WriteRune(ch)

	// Read every subsequent whitespace character into the buffer.
	// Non-whitespace characters and EOF will cause the loop to exit.
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

func (s *Scanner) scanIdent(lookup bool) (tok Token, pos Pos, lit string) {
	// Save the starting position of the identifier.
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

	// If the literal matches a keyword then return that keyword.
	if lookup {
		if tok = Lookup(lit); tok != IDENT {
			return tok, pos, lit
		}
	}
	return IDENT, pos, lit
}

// scanString consumes a contiguous string of non-quote characters.
// Quote characters can be consumed if they're first escaped with a backslash.
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

// ScanString reads a quoted string from a rune reader.
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
			// If the next character is an escape then write the escaped char.
			// If it's not a valid escape then return an error.
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

// ScanBareIdent reads bare identifier from a rune reader.
func ScanBareIdent(r io.RuneScanner) string {
	// Read every ident character into the buffer.
	// Non-ident characters and EOF will cause the loop to exit.
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

// isWhitespace returns true if the rune is a space, tab, or newline.
func isWhitespace(ch rune) bool { return ch == ' ' || ch == '\t' || ch == '\n' }

// isLetter returns true if the rune is a letter.
func isLetter(ch rune) bool { return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') }

// isDigit returns true if the rune is a digit.
func isDigit(ch rune) bool { return (ch >= '0' && ch <= '9') }

// isIdentChar returns true if the rune can be used in an unquoted identifier.
func isIdentChar(ch rune) bool { return isLetter(ch) || isDigit(ch) || ch == '_' }

// isIdentFirstChar returns true if the rune can be used as the first char in an unquoted identifer.
func isIdentFirstChar(ch rune) bool { return isLetter(ch) || ch == '_' }

// scanNumber consumes anything that looks like the start of a number.
func (s *Scanner) scanNumber() (tok Token, pos Pos, lit string) {
	var buf bytes.Buffer

	// Check if the initial rune is a ".".
	ch, pos := s.r.curr()
	if ch == '.' {
		// Peek and see if the next rune is a digit.
		ch1, _ := s.r.read()
		s.r.unread()
		if !isDigit(ch1) {
			return ILLEGAL, pos, "."
		}

		// Unread the full stop so we can read it later.
		s.r.unread()
	} else {
		s.r.unread()
	}

	// Read as many digits as possible.
	_, _ = buf.WriteString(s.scanDigits())

	// If next code points are a full stop and digit then consume them.
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

	// Read as an integer if it doesn't have a fractional part.
	if !isDecimal {
		s.r.unread()
		return INTEGER, pos, buf.String()
	}
	return NUMBER, pos, buf.String()
}

// scanDigits consumes a contiguous series of digits.
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

// skipUntilNewline skips characters until it reaches a newline.
func (s *Scanner) skipUntilNewline() {
	for {
		if ch, _ := s.r.read(); ch == '\n' || ch == eof {
			return
		}
	}
}

// skipUntilEndComment skips characters until it reaches a '*/' symbol.
func (s *Scanner) skipUntilEndComment() error {
	for {
		if ch1, _ := s.r.read(); ch1 == '*' {
			// We might be at the end.
		star:
			ch2, _ := s.r.read()
			if ch2 == '/' {
				return nil
			} else if ch2 == '*' {
				// We are back in the state machine since we see a star.
				goto star
			} else if ch2 == eof {
				return io.EOF
			}
		} else if ch1 == eof {
			return io.EOF
		}
	}
}

// bufScanner represents a wrapper for scanner to add a buffer.
// It provides a fixed-length circular buffer that can be unread.
type bufScanner struct {
	s   *Scanner
	i   int // buffer index
	n   int // buffer size
	buf [3]struct {
		tok Token
		pos Pos
		lit string
	}
}

// newBufScanner returns a new buffered scanner for a reader.
func newBufScanner(r io.Reader) *bufScanner {
	return &bufScanner{s: NewScanner(r)}
}

// Scan reads the next token from the scanner.
func (s *bufScanner) Scan() (tok Token, pos Pos, lit string) {
	// If we have unread tokens then read them off the buffer first.
	if s.n > 0 {
		s.n--
		return s.curr()
	}

	// Move buffer position forward and save the token.
	s.i = (s.i + 1) % len(s.buf)
	buf := &s.buf[s.i]
	buf.tok, buf.pos, buf.lit = s.s.Scan()
	fmt.Printf("Tokens:%s\t\tContent: \"%s\"\n", buf.tok, buf.lit)
	return s.curr()
}

func (s *Scanner) scanRelRange(startPos Pos) (Token, Pos, string) {
	var buf bytes.Buffer
	buf.WriteRune('[')
	buf.WriteRune('*')

	// 持续扫描直到 ]
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

// Unscan pushes the previously token back onto the buffer.
func (s *bufScanner) Unscan() { s.n++ }

// curr returns the last read token.
func (s *bufScanner) curr() (tok Token, pos Pos, lit string) {
	buf := &s.buf[(s.i-s.n+len(s.buf))%len(s.buf)]
	return buf.tok, buf.pos, buf.lit
}

// reader 表示扫描器使用的带缓冲的符文读取器。
// 它提供了一个固定长度的循环缓冲区，可以将字符反读。
type reader struct {
	r   io.RuneScanner
	i   int // 缓冲区索引
	n   int // 未读字符数
	pos Pos // 全局位置跟踪（下一个字符的起始位置）
	buf [3]struct {
		ch  rune
		pos Pos // 字符的起始位置
	}
	eof bool
}

// ReadRune reads the next rune from the reader.
// This is a wrapper function to implement the io.RuneReader interface.
// Note that this function does not return size.
func (r *reader) ReadRune() (ch rune, size int, err error) {
	ch, _ = r.read()
	if ch == eof {
		err = io.EOF
	}
	return
}

// UnreadRune pushes the previously read rune back onto the buffer.
// This is a wrapper function to implement the io.RuneScanner interface.
func (r *reader) UnreadRune() error {
	r.unread()
	return nil
}

// read 从 reader 中读取下一个符文，并返回其起始和结束位置。
func (r *reader) read() (ch rune, pos Pos) {
	// 如果有未读字符，直接返回缓冲区内容
	if r.n > 0 {
		r.n--
		buf := &r.buf[(r.i-r.n+len(r.buf))%len(r.buf)]
		return buf.ch, buf.pos
	}

	// 读取新字符
	var err error
	ch, _, err = r.r.ReadRune()
	if err != nil {
		ch = eof
	}

	// 处理换行符（兼容 Windows 的 \r\n）
	if ch == '\r' {
		if nextCh, _, err := r.r.ReadRune(); err == nil && nextCh == '\n' {
			ch = '\n' // 将 \r\n 转换为 \n
		} else if err == nil {
			_ = r.r.UnreadRune() // 回退非 \n 的字符
		}
	}

	// 记录当前字符的起始位置
	pos = r.pos

	// 更新全局位置（下一个字符的起始位置）
	if ch == '\n' {
		r.pos.Line++
		r.pos.Column = 1
		r.pos.Offset++
	} else if ch != eof {
		r.pos.Column++
		r.pos.Offset++
	}

	// 保存字符和位置到缓冲区
	r.i = (r.i + 1) % len(r.buf)
	buf := &r.buf[r.i]
	buf.ch = ch
	buf.pos = pos

	return ch, pos
}

// unread 它将全局位置回退到上一个字符的起始位置。
func (r *reader) unread() {
	// 防止缓冲区溢出
	if r.n >= len(r.buf) {
		panic("unread buffer overflow")
	}

	// 增加未读计数
	r.n++

	// 只有当存在未读字符时，才回退全局位置
	if r.n > 0 {
		idx := (r.i - r.n + len(r.buf)) % len(r.buf) // 计算缓冲区中当前未读字符的索引
		prevPos := r.buf[idx].pos                    // 获取未读字符的起始位置
		r.pos = prevPos                              // 将全局位置重置为该字符的起始位置

	}
}

// curr returns the last read character and position.
func (r *reader) curr() (ch rune, pos Pos) {
	i := (r.i - r.n + len(r.buf)) % len(r.buf)
	buf := &r.buf[i]
	return buf.ch, buf.pos
}

// eof is a marker code point to signify that the reader can't read any more.
const eof = rune(0)
