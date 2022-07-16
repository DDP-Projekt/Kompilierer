// much of this code was inspired (meaning copied) from craftinginterpreters
package scanner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
	"github.com/kardianos/osext"
)

type Mode uint32

const (
	ModeNone                 = 0           // nothing special
	ModeStrictCapitalization = (1 << iota) // report capitalization errors
	ModeAlias                              // interpret the tokens as alias (enables *arg syntax)
	ModeInitializing                       // allow special characters for inbuilt functions
)

type ErrorHandler func(tok token.Token, msg string)

type Scanner struct {
	file         string // Path to the file
	src          []rune
	errorHandler ErrorHandler // this function is called for all error messages
	mode         Mode         // scanner mode (alias, initializing, ...)

	include       *Scanner            // include directives
	includedFiles map[string]struct{} // files already included are in here

	start            int // start offset of the current token
	cur              int // current read offset
	line             int
	column           int
	indent           int
	shouldIndent     bool // check wether the next whitespace should be counted as indent
	shouldCapitalize bool // check wether the next character should be capitalized
}

// returns a new scanner, or error if one could not be created
// prefers src, but if src is nil it attempts to read the source-code from filePath
func New(filePath string, src []byte, errorHandler ErrorHandler, mode Mode) (*Scanner, error) {
	// default errorHandler does nothing
	if errorHandler == nil {
		errorHandler = func(token.Token, string) {} // to avoid nil pointer dereference
	}

	scan := &Scanner{
		file:             filePath,
		src:              nil,
		errorHandler:     errorHandler,
		mode:             mode,
		include:          nil,
		includedFiles:    make(map[string]struct{}),
		start:            0,
		cur:              0,
		line:             1,
		column:           1,
		indent:           0,
		shouldIndent:     true,
		shouldCapitalize: true,
	}

	// if src is nil filePath is used to load the src from a file
	if src == nil {
		if filepath.Ext(filePath) != ".ddp" {
			scan.errorHandler(token.Token{Line: scan.line, Column: scan.column}, "Der angegebene Pfad ist keine .ddp Datei")
			return nil, errors.New("ungültiger Datei Typ")
		}

		file, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		src = file
		filePath, _ = filepath.Abs(filePath) // if src was loaded from file, add the absolute path to the set, not the one that was passed
		scan.includedFiles[filePath] = struct{}{}
	}

	if !utf8.Valid(src) {
		scan.err("Der Quelltext entspricht nicht dem utf8 Standard")
		return nil, errors.New("invalid utf8 source")
	}

	scan.src = []rune(string(src))

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
	// check if we are currently including a file
	if s.include != nil {
		if tok := s.include.NextToken(); tok.Type == token.EOF {
			s.include = nil
		} else {
			return tok
		}
	}

	s.skipWhitespace()
	s.start = s.cur

	if s.atEnd() {
		return s.newToken(token.EOF)
	}

	char := s.advance()

	if isAlpha(char, s.initMode()) {
		return s.identifier()
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
	case '*':
		if s.aliasMode() {
			return s.aliasParameter()
		}
	}

	return s.errorToken(fmt.Sprintf("Unerwartetes Zeichen '%s'", string(char)))
}

func (s *Scanner) scanEscape(quote rune) bool {
	switch s.peekNext() {
	case 'a', 'b', 'n', 'r', 't', '\\', quote:
		s.advance()
		return true
	default:
		s.err(fmt.Sprintf("Unbekannte Escape Sequenz '\\%v'", s.peekNext()))
		return false
	}
}

func (s *Scanner) string() token.Token {
	for !s.atEnd() {
		if s.peek() == '"' {
			break
		} else if s.peek() == '\n' {
			s.line++
		} else if s.peek() == '\\' {
			s.scanEscape('"')
		}
		s.advance()
	}

	if s.atEnd() {
		return s.errorToken("Offenes Text Literal")
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
			s.line++
		} else if s.peek() == '\\' {
			gotBackslash = true
			s.scanEscape('\'')
		}
		s.advance()
	}

	if s.atEnd() {
		return s.errorToken("Offenes Buchstaben Literal")
	}

	s.advance()
	tok := s.newToken(token.CHAR)
	switch utf8.RuneCountInString(tok.Literal) {
	case 3:
	case 4:
		if !gotBackslash {
			s.err("Ein Buchstaben Literal darf nur einen Buchstaben enthalten")
		}
	default:
		s.err("Ein Buchstaben Literal darf nur einen Buchstaben enthalten")
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

var exe_dir string // path to the folder of the kddp executable

func init() {
	// get the path to the ddp install directory
	if ddppath := os.Getenv("DDPPATH"); ddppath != "" {
		exe_dir = ddppath
	} else if exeFolder, err := osext.ExecutableFolder(); err != nil { // fallback if the environment variable is not set, might fail though
		panic(err)
	} else {
		exe_dir = exeFolder
	}
}

func (s *Scanner) identifier() token.Token {
	shouldReportCapitailzation := false // we don't report capitalization errors on aliases but don't know the tokenType yet, so this flag is used
	if s.strictCapitalizationMode() && s.shouldCapitalize && !isUpper(s.src[s.cur-1]) {
		shouldReportCapitailzation = true
	}

	for isAlphaNumeric(s.peek(), s.initMode()) {
		s.advance()
	}

	tokenType := s.identifierType()

	if shouldReportCapitailzation && tokenType != token.IDENTIFIER {
		s.err("Nach einem Punkt muss ein Großbuchstabe folgen") // not a critical error, so continue and let the error handler to the job
	}

	if tokenType == token.BINDE && !s.aliasMode() { // don't resolve includes in alias mode (they would lead to garbage anyways)
		lit := s.NextToken()
		if lit.Type != token.STRING {
			s.err("Nach 'Binde' muss ein Text Literal folgen")
			return lit
		}

		if s.NextToken().Type != token.EIN {
			s.err("Es wurde 'ein' erwartet")
		} else if s.NextToken().Type != token.DOT {
			s.err("Nach 'ein' muss ein Punkt folgen")
		}

		literalContent := strings.Trim(lit.Literal, "\"")
		inclPath := ""
		var err error
		if filepath.Dir(literalContent) == "Duden" {
			inclPath = filepath.Join(exe_dir, literalContent) + ".ddp"
		} else {
			inclPath, err = filepath.Abs(literalContent + ".ddp") // to eliminate ambiguity with nested includes
		}
		if err != nil {
			s.err(fmt.Sprintf("Fehler beim Einbinden der Datei '%s': \"%s\"", literalContent+".ddp", err.Error()))
		} else if _, ok := s.includedFiles[inclPath]; !ok {
			if s.include, err = New(inclPath, nil, s.errorHandler, s.mode); err != nil {
				s.err(fmt.Sprintf("Fehler beim Einbinden der Datei '%s': \"%s\"", inclPath, err.Error()))
			} else {
				// append the already included files
				for k, v := range s.includedFiles {
					s.include.includedFiles[k] = v
				}
			}
		}

		return s.NextToken()
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

// helper to scan the *argname in aliases
func (s *Scanner) aliasParameter() token.Token {
	for isAlphaNumeric(s.peek(), s.initMode()) {
		s.advance()
	}

	if tokenType := s.identifierType(); tokenType != token.IDENTIFIER {
		s.err("Es wurde ein Name als Alias-Parameter erwartet")
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
			s.line++
			s.indent = 0
			s.column = 0
			s.shouldIndent = true
			s.advance()
		case '[':
			s.advance()
			bracketCount := 1

			for bracketCount > 0 && !s.atEnd() {
				switch s.peek() {
				case '[':
					bracketCount++
				case ']':
					bracketCount--
				case '\n':
					s.line++
				}
				s.advance()
			}
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
		File:      s.file,
		Line:      s.line,
		Column:    s.column - (s.cur - s.start),
		AliasInfo: nil,
	}
}

func (s *Scanner) errorToken(msg string) token.Token {
	s.err(msg)
	return token.Token{
		Type:      token.ILLEGAL,
		Literal:   msg,
		File:      s.file,
		Line:      s.line,
		Column:    s.column,
		AliasInfo: nil,
	}
}

const eof = -1

func (s *Scanner) advance() rune {
	s.cur++
	s.column++
	if s.shouldIndent && !isSpace(s.src[s.cur-1]) {
		s.shouldIndent = false
	}
	return s.src[s.cur-1]
}

func (s *Scanner) peek() rune {
	if s.atEnd() {
		return eof
	}
	return s.src[s.cur]
}

func (s *Scanner) peekNext() rune {
	if s.atEnd() || s.cur+1 >= len(s.src) {
		return eof
	}
	return s.src[s.cur+1]
}

func (s *Scanner) err(msg string) {
	tok := token.Token{
		File:    s.file,
		Line:    s.line,
		Column:  s.column,
		Indent:  s.indent,
		Literal: msg,
	}
	if s.aliasMode() {
		s.errorHandler(tok, fmt.Sprintf("Fehler im Alias '%s': %s", string(s.src), msg))
	} else {
		s.errorHandler(tok, msg)
	}
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

func (s *Scanner) initMode() bool {
	return s.mode&ModeInitializing != 0
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isAlpha(r rune, initializing bool) bool {
	if initializing && r == '§' {
		return true
	}
	return ('a' <= r && r <= 'z') ||
		('A' <= r && r <= 'Z') ||
		r == 'ß' || r == '_' || r == 'ä' ||
		r == 'Ä' || r == 'ö' || r == 'Ö' ||
		r == 'ü' || r == 'Ü'
}

func isAlphaNumeric(r rune, initializing bool) bool {
	return isAlpha(r, initializing) || isDigit(r)
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\r' || r == '\n' || r == '\t'
}

func isUpper(r rune) bool {
	return ('A' <= r && r <= 'Z') ||
		r == 'Ä' || r == 'Ü' || r == 'Ö'
}
