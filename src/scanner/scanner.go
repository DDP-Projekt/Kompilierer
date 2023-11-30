// much of this code was inspired (meaning copied) from craftinginterpreters
package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type Mode uint32

const (
	ModeNone                 = 0           // nothing special
	ModeStrictCapitalization = (1 << iota) // report capitalization errors
	ModeAlias                              // interpret the tokens as alias (enables *arg syntax)
)

type Scanner struct {
	file         string // Path to the file
	src          []byte
	errorHandler ddperror.Handler // this function is called for all error messages
	mode         Mode             // scanner mode (alias, initializing, ...)

	start            int // start offset of the current token
	cur              int // current read offset
	line             uint
	column           uint
	startLine        uint // to construct valid ranges
	startColumn      uint // to construct valid ranges
	indent           uint
	shouldIndent     bool // check wether the next whitespace should be counted as indent
	shouldCapitalize bool // check wether the next character should be capitalized
}

// returns a new scanner, or error if one could not be created
// prefers src, but if src is nil it attempts to read the source-code from filePath
func New(filePath string, src []byte, errorHandler ddperror.Handler, mode Mode) (*Scanner, error) {
	// default errorHandler does nothing
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
	}

	scan := &Scanner{
		file:             filePath,
		src:              nil,
		errorHandler:     errorHandler,
		mode:             mode,
		start:            0,
		cur:              0,
		line:             1,
		column:           1,
		startLine:        1,
		startColumn:      1,
		indent:           0,
		shouldIndent:     true,
		shouldCapitalize: true,
	}

	// if src is nil filePath is used to load the src from a file
	if src == nil {
		if filepath.Ext(filePath) != ".ddp" {
			return nil, ddperror.New(ddperror.SYN_MALFORMED_INCLUDE_PATH, scan.currentRange(), ddperror.MSG_INVALID_FILE_EXTENSION, scan.file)
		}

		file, err := os.ReadFile(filePath)
		if err != nil {
			return nil, ddperror.New(ddperror.MISC_INCLUDE_ERROR, scan.currentRange(), err.Error(), scan.file)
		}

		src = file
	}

	if !utf8.Valid(src) {
		return nil, ddperror.New(ddperror.SYN_INVALID_UTF8, scan.currentRange(), ddperror.MSG_INVALID_UTF8, scan.file)
	}

	scan.src = src

	return scan, nil
}

// scan all tokens in the scanners source until EOF occurs
func (s *Scanner) ScanAll() []token.Token {
	tokens := make([]token.Token, 0)
	var tok token.Token
	for tok = s.NextToken(); tok.Type != token.EOF; tok = s.NextToken() {
		tokens = append(tokens, tok)
	}

	tokens = append(tokens, tok) // append the EOF
	return tokens
}

// scan the next token from source
// if all tokens were scanned it returns EOF
func (s *Scanner) NextToken() token.Token {
	s.skipWhitespace()
	s.start, s.startLine, s.startColumn = s.cur, s.line, s.column

	if s.atEnd() {
		return s.newToken(token.EOF)
	}

	char := s.advance()

	if isAlpha(char) {
		return s.identifier(char)
	}
	if isDigit(char) {
		return s.number()
	}

	switch char {
	case '-':
		return s.newToken(token.NEGATE)
	case '.':
		return s.newToken(token.DOT)
	case ',':
		return s.newToken(token.COMMA)
	case ':':
		return s.newToken(token.COLON)
	case '(':
		return s.newToken(token.LPAREN)
	case ')':
		return s.newToken(token.RPAREN)
	case '"':
		return s.string()
	case '\'':
		return s.char()
	case '[':
		bracketCount := 1

		for bracketCount > 0 && !s.atEnd() {
			switch s.peek() {
			case '[':
				bracketCount++
			case ']':
				bracketCount--
			case '\n':
				s.increaseLineBeforeAdvance()
			}
			s.advance()
		}
		return s.newToken(token.COMMENT)
	case '<':
		if s.aliasMode() {
			return s.aliasParameter()
		}
	}

	return s.newToken(token.SYMBOL)
}

func (s *Scanner) scanEscape(quote rune) bool {
	switch s.peekNext() {
	case 'a', 'b', 'n', 'r', 't', '\\', quote:
		s.advance()
		return true
	default:
		s.err(
			ddperror.SYN_MALFORMED_LITERAL,
			token.Range{
				Start: token.Position{
					Line:   s.line,
					Column: s.column,
				},
				End: token.Position{
					Line:   s.line,
					Column: s.column + 2,
				},
			},
			fmt.Sprintf("Unbekannte Escape Sequenz '\\%v'", s.peekNext()),
		)
		return false
	}
}

func (s *Scanner) string() token.Token {
	for !s.atEnd() {
		if s.peek() == '"' {
			break
		} else if s.peek() == '\n' {
			s.increaseLineBeforeAdvance()
		} else if s.peek() == '\\' {
			s.scanEscape('"')
		}
		s.advance()
	}

	if s.atEnd() {
		return s.errorToken("ein Offenes Text Literal")
	}

	s.advance()
	return s.newToken(token.STRING)
}

func (s *Scanner) char() token.Token {
	gotBackslash := false
	for !s.atEnd() {
		if s.peek() == '\'' {
			break
		} else if s.peek() == '\n' {
			s.increaseLineBeforeAdvance()
		} else if s.peek() == '\\' {
			gotBackslash = true
			s.scanEscape('\'')
		}
		s.advance()
	}

	if s.atEnd() {
		return s.errorToken("ein Offenes Buchstaben Literal")
	}

	s.advance()
	tok := s.newToken(token.CHAR)
	switch utf8.RuneCountInString(tok.Literal) {
	case 3:
	case 4:
		if !gotBackslash {
			s.err(ddperror.SYN_MALFORMED_LITERAL, tok.Range, ddperror.MSG_CHAR_LITERAL_TOO_LARGE)
		}
	default:
		s.err(ddperror.SYN_MALFORMED_LITERAL, tok.Range, ddperror.MSG_CHAR_LITERAL_TOO_LARGE)
	}
	return tok
}

func (s *Scanner) number() token.Token {
	tok := token.INT
	for isDigit(s.peek()) {
		s.advance()
	}

	if s.peek() == ',' && isDigit(s.peekNext()) {
		tok = token.FLOAT
		s.advance()
		for isDigit(s.peek()) {
			s.advance()
		}
	}

	return s.newToken(tok)
}

func (s *Scanner) identifier(start rune) token.Token {
	shouldReportCapitailzation := false // we don't report capitalization errors on aliases but don't know the tokenType yet, so this flag is used
	var capitalRange token.Range
	if s.strictCapitalizationMode() && s.shouldCapitalize && !isUpper(start) {
		shouldReportCapitailzation = true
		capitalRange = s.currentRange()
	}

	for isAlphaNumeric(s.peek()) {
		s.advance()
	}

	tokenType := s.identifierType()

	if shouldReportCapitailzation && tokenType != token.IDENTIFIER {
		s.err(ddperror.SYN_EXPECTED_CAPITAL, capitalRange, "Nach einem Punkt muss ein Großbuchstabe folgen") // not a critical error, so continue and let the error handler to the job
	}

	return s.newToken(tokenType)
}

func (s *Scanner) identifierType() token.TokenType {
	lit := string(s.src[s.start:s.cur])

	tokenType := token.KeywordToTokenType(lit)
	if tokenType == token.IDENTIFIER {
		litTokenType := token.KeywordToTokenType(strings.ToLower(lit))
		if litTokenType != tokenType {
			tokenType = litTokenType
		}
	}

	return tokenType
}

// helper to scan the <argname> in aliases
func (s *Scanner) aliasParameter() token.Token {
	if !isAlpha(s.peek()) {
		s.err(ddperror.SYN_MALFORMED_ALIAS, s.currentRange(), "Invalider Parameter Name")
	}
	for !s.atEnd() && s.peek() != '>' {
		if !isAlphaNumeric(s.advance()) {
			s.err(ddperror.SYN_MALFORMED_ALIAS, s.currentRange(), "Invalider Parameter Name")
		}
	}
	if s.atEnd() {
		s.err(ddperror.SYN_MALFORMED_ALIAS, s.currentRange(), "Offener Parameter")
	} else {
		s.advance() // consume the closing >
	}
	if s.cur-s.start <= 2 && !s.atEnd() {
		s.err(ddperror.SYN_MALFORMED_ALIAS, s.currentRange(), "Ein Parameter in einem Alias muss mindestens einen Buchstaben enthalten")
	}

	if tokenType := s.identifierType(); tokenType != token.IDENTIFIER {
		s.err(ddperror.SYN_MALFORMED_ALIAS, s.currentRange(), "Es wurde ein Name als Alias-Parameter erwartet")
	}

	return s.newToken(token.ALIAS_PARAMETER)
}

func (s *Scanner) skipWhitespace() {
	consecutiveSpaceCount := 0
	for {
		char := s.peek()
		if char == ' ' {
			consecutiveSpaceCount++
		} else {
			consecutiveSpaceCount = 0
		}

		switch char {
		case ' ':
			if s.shouldIndent && consecutiveSpaceCount == 4 {
				s.indent++
				consecutiveSpaceCount = 0
			}
			s.advance()
		case '\r':
			s.advance()
		case '\t':
			if s.shouldIndent {
				s.indent++
			}
			s.advance()
		case '\n':
			s.increaseLineBeforeAdvance()
			s.advance()
		default:
			return
		}
	}
}

func (s *Scanner) atEnd() bool {
	return s.cur >= len(s.src)
}

func (s *Scanner) newToken(tokenType token.TokenType) token.Token {
	if tokenType == token.DOT || tokenType == token.COLON {
		s.shouldCapitalize = true
	} else {
		s.shouldCapitalize = false
	}

	return token.Token{
		Type:      tokenType,
		Literal:   string(s.src[s.start:s.cur]),
		Indent:    s.indent,
		Range:     s.currentRange(),
		AliasInfo: nil,
	}
}

func (s *Scanner) errorToken(msg string) token.Token {
	return token.Token{
		Type:      token.ILLEGAL,
		Literal:   msg,
		Indent:    s.indent,
		Range:     s.currentRange(),
		AliasInfo: nil,
	}
}

func (s *Scanner) currentRange() token.Range {
	return token.Range{
		Start: token.Position{
			Line:   s.startLine,
			Column: s.startColumn,
		},
		End: token.Position{
			Line:   s.line,
			Column: s.column,
		},
	}
}

const eof = -1

func (s *Scanner) advance() rune {
	r, w := utf8.DecodeRune(s.src[s.cur:])
	s.cur += w
	s.column++
	if s.shouldIndent && !isSpace(r) {
		s.shouldIndent = false
	}
	return r
}

func (s *Scanner) peek() rune {
	if s.atEnd() {
		return eof
	}
	r, _ := utf8.DecodeRune(s.src[s.cur:])
	return r
}

func (s *Scanner) peekNext() rune {
	if s.atEnd() || s.cur+1 >= len(s.src) {
		return eof
	}
	_, w := utf8.DecodeRune(s.src[s.cur:])
	r, _ := utf8.DecodeRune(s.src[s.cur+w:])
	return r
}

func (s *Scanner) err(code ddperror.Code, Range token.Range, msg string) {
	e := ddperror.New(code, Range, msg, s.file)
	if s.aliasMode() {
		e.Msg = fmt.Sprintf("Fehler im Alias '%s': %s", string(s.src), e.Msg)
	}
	s.errorHandler(e)
}

func (s *Scanner) increaseLineBeforeAdvance() {
	s.line++
	s.indent = 0
	s.column = 0 // will be increased in advance()
	s.shouldIndent = true
}

func (s *Scanner) Mode() Mode {
	return s.mode
}

func (s *Scanner) strictCapitalizationMode() bool {
	return s.mode&ModeStrictCapitalization != 0
}

func (s *Scanner) aliasMode() bool {
	return s.mode&ModeAlias != 0
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isAlpha(r rune) bool {
	return ('a' <= r && r <= 'z') ||
		('A' <= r && r <= 'Z') ||
		r == 'ß' || r == '_' || r == 'ä' ||
		r == 'Ä' || r == 'ö' || r == 'Ö' ||
		r == 'ü' || r == 'Ü'
}

func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\r' || r == '\n' || r == '\t'
}

func isUpper(r rune) bool {
	return ('A' <= r && r <= 'Z') ||
		r == 'Ä' || r == 'Ü' || r == 'Ö'
}
