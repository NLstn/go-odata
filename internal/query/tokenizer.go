package query

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenString
	TokenNumber
	TokenBoolean
	TokenNull
	TokenOperator
	TokenLogical
	TokenNot
	TokenLParen
	TokenRParen
	TokenComma
	TokenArithmetic
	TokenColon
	TokenDate
	TokenTime
	TokenDateTime
	TokenGUID
)

// Token represents a single token in the filter expression
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Tokenizer tokenizes OData filter expressions
type Tokenizer struct {
	input       string
	pos         int
	ch          rune
	tokenBuffer []Token // Pre-allocated buffer for tokens
	tokenIndex  int     // Current position in token buffer
}

// tokenizerPool is a sync.Pool for reusing Tokenizer instances.
// This reduces allocations by ~10-14% by avoiding buffer allocation per tokenization.
var tokenizerPool = sync.Pool{
	New: func() interface{} {
		return &Tokenizer{
			tokenBuffer: make([]Token, 0, minTokenSliceCapacity),
		}
	},
}

// AcquireTokenizer gets a Tokenizer from the pool and initializes it with the input
func AcquireTokenizer(input string) *Tokenizer {
	t := tokenizerPool.Get().(*Tokenizer)
	t.input = input
	t.pos = 0
	t.tokenIndex = 0
	// Reset the slice length but keep capacity
	t.tokenBuffer = t.tokenBuffer[:0]

	// Ensure sufficient capacity for expected tokens
	estimatedTokens := len(input)/estimatedAvgTokenLength + 1
	if estimatedTokens < minTokenSliceCapacity {
		estimatedTokens = minTokenSliceCapacity
	}
	if cap(t.tokenBuffer) < estimatedTokens {
		t.tokenBuffer = make([]Token, 0, estimatedTokens)
	}

	if len(input) > 0 {
		t.ch = rune(input[0])
	} else {
		t.ch = 0
	}
	return t
}

// ReleaseTokenizer returns a Tokenizer to the pool
func ReleaseTokenizer(t *Tokenizer) {
	if t == nil {
		return
	}
	// Clear input reference to allow garbage collection
	t.input = ""
	t.ch = 0
	// Keep the token buffer capacity for reuse (up to a reasonable limit)
	// If buffer is too large, let it be garbage collected
	if cap(t.tokenBuffer) <= 256 {
		tokenizerPool.Put(t)
	}
}

// NewTokenizer creates a new tokenizer
func NewTokenizer(input string) *Tokenizer {
	// Pre-allocate token buffer based on estimated number of tokens
	estimatedTokens := len(input)/estimatedAvgTokenLength + 1
	if estimatedTokens < minTokenSliceCapacity {
		estimatedTokens = minTokenSliceCapacity
	}

	t := &Tokenizer{
		input:       input,
		pos:         0,
		tokenBuffer: make([]Token, 0, estimatedTokens),
		tokenIndex:  0,
	}
	if len(input) > 0 {
		t.ch = rune(input[0])
	}
	return t
}

// getToken gets a token from the pre-allocated buffer and initializes it
func (t *Tokenizer) getToken(typ TokenType, value string, pos int) *Token {
	// Check if we need to grow the buffer
	if t.tokenIndex >= cap(t.tokenBuffer) {
		// Grow the buffer by 50%
		newCap := cap(t.tokenBuffer) + cap(t.tokenBuffer)/2
		if newCap < minTokenSliceCapacity {
			newCap = minTokenSliceCapacity
		}
		newBuffer := make([]Token, t.tokenIndex, newCap)
		copy(newBuffer, t.tokenBuffer)
		t.tokenBuffer = newBuffer
	}

	// Extend slice if needed
	if t.tokenIndex >= len(t.tokenBuffer) {
		t.tokenBuffer = t.tokenBuffer[:t.tokenIndex+1]
	}

	// Reuse token from buffer
	tok := &t.tokenBuffer[t.tokenIndex]
	tok.Type = typ
	tok.Value = value
	tok.Pos = pos
	t.tokenIndex++
	return tok
}

// advance moves to the next character
func (t *Tokenizer) advance() {
	t.pos++
	if t.pos >= len(t.input) {
		t.ch = 0 // EOF
	} else {
		t.ch = rune(t.input[t.pos])
	}
}

// peek looks ahead without advancing
func (t *Tokenizer) peek() rune {
	if t.pos+1 >= len(t.input) {
		return 0
	}
	return rune(t.input[t.pos+1])
}

// skipWhitespace skips whitespace characters
func (t *Tokenizer) skipWhitespace() {
	for t.ch == ' ' || t.ch == '\t' || t.ch == '\n' || t.ch == '\r' {
		t.advance()
	}
}

// readString reads a quoted string
// Per OData v4 spec, single quotes within string literals are escaped by doubling them (")
func (t *Tokenizer) readString() string {
	quote := t.ch
	t.advance() // skip opening quote

	start := t.pos
	hasEscapes := false

	// Fast path: scan for simple strings without escape sequences
	for t.ch != 0 {
		if t.ch == quote {
			if t.peek() == quote {
				// Found an escaped quote - need to use slow path
				hasEscapes = true
				break
			}
			// This is the closing quote
			result := t.input[start:t.pos]
			t.advance() // skip closing quote
			return result
		}
		t.advance()
	}

	// Slow path: handle escape sequences
	if hasEscapes {
		var result strings.Builder
		// Pre-size buffer: content so far plus extra for potential escape sequence growth
		result.Grow(t.pos - start + stringBuilderExtraCapacity)
		// Write what we've read so far
		result.WriteString(t.input[start:t.pos])

		for t.ch != 0 {
			if t.ch == quote {
				if t.peek() == quote {
					// This is an escaped quote - add one quote to result and skip both
					result.WriteRune(quote)
					t.advance() // skip first quote
					t.advance() // skip second quote
				} else {
					// This is the closing quote
					break
				}
			} else {
				result.WriteRune(t.ch)
				t.advance()
			}
		}

		if t.ch == quote {
			t.advance() // skip closing quote
		}
		return result.String()
	}

	// String ended without closing quote
	return t.input[start:t.pos]
}

// readNumber reads a number using substring slicing to avoid allocations
func (t *Tokenizer) readNumber() string {
	start := t.pos

	// Handle negative numbers
	if t.ch == '-' {
		t.advance()
	}

	// Read integer part
	for unicode.IsDigit(t.ch) {
		t.advance()
	}

	// Read decimal part
	if t.ch == '.' {
		t.advance()
		for unicode.IsDigit(t.ch) {
			t.advance()
		}
	}

	// Read exponent part
	if t.ch == 'e' || t.ch == 'E' {
		t.advance()
		if t.ch == '+' || t.ch == '-' {
			t.advance()
		}
		for unicode.IsDigit(t.ch) {
			t.advance()
		}
	}

	// Return a slice of the input string to avoid allocation
	return t.input[start:t.pos]
}

// isIdentifierChar checks if a character is valid within an identifier (ASCII fast path)
func isIdentifierChar(ch rune) bool {
	// Fast path for common ASCII characters
	if ch <= 127 {
		return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '.'
	}
	// Fallback to unicode for non-ASCII
	return unicode.IsLetter(ch) || unicode.IsDigit(ch)
}

// isIdentifierStart checks if a character can start an identifier (ASCII fast path)
func isIdentifierStart(ch rune) bool {
	// Fast path for common ASCII characters
	if ch <= 127 {
		return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '$'
	}
	// Fallback to unicode for non-ASCII
	return unicode.IsLetter(ch)
}

// readIdentifier reads an identifier or keyword
// In OData v4, identifiers can contain dots for qualified names (e.g., Edm.String)
func (t *Tokenizer) readIdentifier() string {
	start := t.pos

	// Allow $ at the beginning for special properties like $count
	if t.ch == '$' {
		t.advance()
	}

	// Fast path: use direct slice access for ASCII-only identifiers
	for t.ch != 0 && isIdentifierChar(t.ch) {
		t.advance()
	}

	// Return a slice of the input string to avoid allocation
	return t.input[start:t.pos]
}

// isDateLiteral checks if current position starts a date literal (YYYY-MM-DD)
func (t *Tokenizer) isDateLiteral() bool {
	// Look ahead to check for date pattern: 4 digits, dash, 2 digits, dash, 2 digits
	if t.pos+10 > len(t.input) {
		return false
	}

	// Check pattern: DDDD-DD-DD
	str := t.input[t.pos : t.pos+10]
	if len(str) != 10 {
		return false
	}

	// Check for YYYY-MM-DD format
	for i, ch := range str {
		if i == 4 || i == 7 {
			if ch != '-' {
				return false
			}
		} else {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}

	return true
}

// readDateLiteral reads a date literal (YYYY-MM-DD) using substring slicing
func (t *Tokenizer) readDateLiteral() string {
	start := t.pos

	// Read YYYY-MM-DD (10 characters)
	for i := 0; i < 10 && t.ch != 0; i++ {
		t.advance()
	}

	// Return a slice of the input string to avoid allocation
	return t.input[start:t.pos]
}

// isTimeLiteral checks if current position starts a time literal (HH:MM:SS or HH:MM:SS.sss)
func (t *Tokenizer) isTimeLiteral() bool {
	// Look ahead to check for time pattern: 2 digits, colon, 2 digits, colon, 2 digits
	if t.pos+8 > len(t.input) {
		return false
	}

	// Check pattern: DD:DD:DD (minimum)
	str := t.input[t.pos:]
	if len(str) < 8 {
		return false
	}

	// Check for HH:MM:SS format
	for i := 0; i < 8; i++ {
		ch := str[i]
		if i == 2 || i == 5 {
			if ch != ':' {
				return false
			}
		} else {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}

	return true
}

// readTimeLiteral reads a time literal (HH:MM:SS or HH:MM:SS.sss) using substring slicing
func (t *Tokenizer) readTimeLiteral() string {
	start := t.pos

	// Read HH:MM:SS (8 characters)
	for i := 0; i < 8 && t.ch != 0; i++ {
		t.advance()
	}

	// Read optional fractional seconds (.sss...)
	if t.ch == '.' {
		t.advance()
		for unicode.IsDigit(t.ch) {
			t.advance()
		}
	}

	// Return a slice of the input string to avoid allocation
	return t.input[start:t.pos]
}

// isGUIDLiteral checks if current position starts a GUID literal (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
// GUID format: 8 hex chars, dash, 4 hex chars, dash, 4 hex chars, dash, 4 hex chars, dash, 12 hex chars
func (t *Tokenizer) isGUIDLiteral() bool {
	// Look ahead to check for GUID pattern: 8-4-4-4-12 = 36 characters total
	if t.pos+36 > len(t.input) {
		return false
	}

	str := t.input[t.pos : t.pos+36]
	if len(str) != 36 {
		return false
	}

	// Check for proper GUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// Positions of dashes: 8, 13, 18, 23
	for i, ch := range str {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if ch != '-' {
				return false
			}
		} else {
			// Must be a hex digit
			if !isHexDigit(byte(ch)) {
				return false
			}
		}
	}

	// Check that the character after the GUID is not alphanumeric (to avoid partial matches)
	if t.pos+36 < len(t.input) {
		nextChar := t.input[t.pos+36]
		if isHexDigit(nextChar) || nextChar == '-' {
			return false
		}
	}

	return true
}

// isHexDigit checks if a byte is a valid hexadecimal digit
func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F')
}

// readGUIDLiteral reads a GUID literal (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx) using substring slicing
func (t *Tokenizer) readGUIDLiteral() string {
	start := t.pos

	// Read the 36 character GUID
	for i := 0; i < 36 && t.ch != 0; i++ {
		t.advance()
	}

	// Return a slice of the input string to avoid allocation
	return t.input[start:t.pos]
}
func (t *Tokenizer) NextToken() (*Token, error) {
	t.skipWhitespace()

	if t.ch == 0 {
		return t.getToken(TokenEOF, "", t.pos), nil
	}

	pos := t.pos

	// Try to tokenize based on character type
	if token := t.tokenizeString(pos); token != nil {
		return token, nil
	}

	if token := t.tokenizeNumber(pos); token != nil {
		return token, nil
	}

	if token := t.tokenizeSpecialChar(pos); token != nil {
		return token, nil
	}

	if token := t.tokenizeIdentifierOrKeyword(pos); token != nil {
		return token, nil
	}

	return nil, fmt.Errorf("unexpected character '%c' at position %d", t.ch, t.pos)
}

// tokenizeString tokenizes string literals
func (t *Tokenizer) tokenizeString(pos int) *Token {
	if t.ch == '\'' || t.ch == '"' {
		value := t.readString()
		return t.getToken(TokenString, value, pos)
	}
	return nil
}

// tokenizeNumber tokenizes numeric literals, date/time literals, or GUID literals
func (t *Tokenizer) tokenizeNumber(pos int) *Token {
	if unicode.IsDigit(t.ch) {
		// Check for GUID first using strict validation (8-4-4-4-12 hex pattern)
		// GUIDs are validated strictly to prevent confusion with dates
		// e.g., "12345678-1234-1234-1234-123456789012" is a valid GUID
		// e.g., "2024-01-01" will fail GUID check and be handled as a date
		if t.isGUIDLiteral() {
			value := t.readGUIDLiteral()
			return t.getToken(TokenGUID, value, pos)
		}
		if literal, ok := t.readDateTimeLiteralIfPresent(); ok {
			return t.getToken(TokenDateTime, literal, pos)
		}
		if t.isDateLiteral() {
			value := t.readDateLiteral()
			return t.getToken(TokenDate, value, pos)
		}
		if t.isTimeLiteral() {
			value := t.readTimeLiteral()
			return t.getToken(TokenTime, value, pos)
		}
	}

	// Otherwise parse as number
	if unicode.IsDigit(t.ch) || (t.ch == '-' && unicode.IsDigit(t.peek())) {
		value := t.readNumber()
		return t.getToken(TokenNumber, value, pos)
	}
	return nil
}

// readDateTimeLiteralIfPresent detects and reads a full datetime literal (YYYY-MM-DDThh:mm:ss[.fff][Z|+/-hh:mm])
func (t *Tokenizer) readDateTimeLiteralIfPresent() (string, bool) {
	if !t.isDateLiteral() {
		return "", false
	}

	start := t.pos
	datePart := t.readDateLiteral()
	if t.ch != 'T' && t.ch != 't' {
		t.pos = start
		t.ch = rune(t.input[t.pos])
		return "", false
	}

	// Include the 'T'
	t.advance()

	if !t.isTimeLiteral() {
		t.pos = start
		t.ch = rune(t.input[t.pos])
		return "", false
	}

	timePart := t.readTimeLiteral()

	var builder strings.Builder
	builder.WriteString(datePart)
	builder.WriteByte('T')
	builder.WriteString(timePart)

	// Optional fractional seconds already handled inside readTimeLiteral
	// Handle timezone designator (Z or +/-hh:mm)
	switch t.ch {
	case 'Z', 'z':
		builder.WriteByte('Z')
		t.advance()
	case '+', '-':
		builder.WriteRune(t.ch)
		t.advance()
		for i := 0; i < 2 && unicode.IsDigit(t.ch); i++ {
			builder.WriteRune(t.ch)
			t.advance()
		}
		if t.ch == ':' {
			builder.WriteByte(':')
			t.advance()
		}
		for i := 0; i < 2 && unicode.IsDigit(t.ch); i++ {
			builder.WriteRune(t.ch)
			t.advance()
		}
	}

	return builder.String(), true
}

// tokenizeSpecialChar tokenizes special characters (parentheses, comma, operators)
func (t *Tokenizer) tokenizeSpecialChar(pos int) *Token {
	switch t.ch {
	case '(':
		t.advance()
		return t.getToken(TokenLParen, "(", pos)
	case ')':
		t.advance()
		return t.getToken(TokenRParen, ")", pos)
	case ',':
		t.advance()
		return t.getToken(TokenComma, ",", pos)
	case ':':
		t.advance()
		return t.getToken(TokenColon, ":", pos)
	case '+', '-', '*', '/':
		op := string(t.ch)
		t.advance()
		return t.getToken(TokenArithmetic, op, pos)
	}
	return nil
}

// toLowerASCII converts an ASCII byte to lowercase without allocation
func toLowerASCII(ch byte) byte {
	if ch >= 'A' && ch <= 'Z' {
		return ch + 32 // Convert to lowercase
	}
	return ch
}

// equalsFoldASCII compares two strings for equality, ignoring ASCII case.
// The target string MUST be lowercase for correct comparison.
// This is more efficient than strings.EqualFold for short ASCII strings
// because it avoids Unicode handling overhead.
func equalsFoldASCII(s, target string) bool {
	if len(s) != len(target) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if toLowerASCII(s[i]) != target[i] {
			return false
		}
	}
	return true
}

// tokenizeIdentifierOrKeyword tokenizes identifiers and keywords
func (t *Tokenizer) tokenizeIdentifierOrKeyword(pos int) *Token {
	// Allow identifiers starting with letters or $ (for special properties like $count)
	if !isIdentifierStart(t.ch) {
		return nil
	}

	// Check if this could be a GUID starting with a hex letter (a-f, A-F)
	// GUIDs can start with letters and look like: abcdef12-3456-7890-abcd-ef1234567890
	if t.isGUIDLiteral() {
		value := t.readGUIDLiteral()
		return t.getToken(TokenGUID, value, pos)
	}

	value := t.readIdentifier()

	// Check for arithmetic functions: add, sub, mul, div, mod can be either
	// functions (when followed by '(') or infix operators
	// Use fast ASCII case-insensitive comparison
	if t.ch == '(' {
		if equalsFoldASCII(value, "add") || equalsFoldASCII(value, "sub") ||
			equalsFoldASCII(value, "mul") || equalsFoldASCII(value, "div") ||
			equalsFoldASCII(value, "mod") || equalsFoldASCII(value, "has") {
			// Treat as identifier (function name) when followed by '('
			return t.getToken(TokenIdentifier, value, pos)
		}
	}

	// Check for keywords using fast ASCII comparison
	if token := t.classifyKeywordFast(value, pos); token != nil {
		return token
	}

	// Functions like contains, startswith, endswith are identifiers
	// that will be recognized as function calls when followed by '('
	return t.getToken(TokenIdentifier, value, pos)
}

// classifyKeywordFast classifies a keyword using fast ASCII case-insensitive comparison
// Returns the appropriate token or nil if not a keyword
func (t *Tokenizer) classifyKeywordFast(value string, pos int) *Token {
	// Use length as first discriminator for efficiency
	switch len(value) {
	case 2:
		if equalsFoldASCII(value, "or") {
			return t.getToken(TokenLogical, "or", pos)
		}
		if equalsFoldASCII(value, "eq") {
			return t.getToken(TokenOperator, "eq", pos)
		}
		if equalsFoldASCII(value, "ne") {
			return t.getToken(TokenOperator, "ne", pos)
		}
		if equalsFoldASCII(value, "gt") {
			return t.getToken(TokenOperator, "gt", pos)
		}
		if equalsFoldASCII(value, "ge") {
			return t.getToken(TokenOperator, "ge", pos)
		}
		if equalsFoldASCII(value, "lt") {
			return t.getToken(TokenOperator, "lt", pos)
		}
		if equalsFoldASCII(value, "le") {
			return t.getToken(TokenOperator, "le", pos)
		}
		if equalsFoldASCII(value, "in") {
			return t.getToken(TokenOperator, "in", pos)
		}
	case 3:
		if equalsFoldASCII(value, "and") {
			return t.getToken(TokenLogical, "and", pos)
		}
		if equalsFoldASCII(value, "not") {
			return t.getToken(TokenNot, "not", pos)
		}
		if equalsFoldASCII(value, "has") {
			return t.getToken(TokenOperator, "has", pos)
		}
		if equalsFoldASCII(value, "add") {
			return t.getToken(TokenArithmetic, "add", pos)
		}
		if equalsFoldASCII(value, "sub") {
			return t.getToken(TokenArithmetic, "sub", pos)
		}
		if equalsFoldASCII(value, "mul") {
			return t.getToken(TokenArithmetic, "mul", pos)
		}
		if equalsFoldASCII(value, "div") {
			return t.getToken(TokenArithmetic, "div", pos)
		}
		if equalsFoldASCII(value, "mod") {
			return t.getToken(TokenArithmetic, "mod", pos)
		}
		if equalsFoldASCII(value, "inf") {
			return t.getToken(TokenNumber, "inf", pos)
		}
		if equalsFoldASCII(value, "nan") {
			return t.getToken(TokenNumber, "nan", pos)
		}
	case 4:
		if equalsFoldASCII(value, "true") {
			return t.getToken(TokenBoolean, "true", pos)
		}
		if equalsFoldASCII(value, "null") {
			return t.getToken(TokenNull, "null", pos)
		}
	case 5:
		if equalsFoldASCII(value, "false") {
			return t.getToken(TokenBoolean, "false", pos)
		}
	}
	return nil
}

const (
	// estimatedAvgTokenLength is the estimated average length of a token in characters.
	// OData tokens like "and", "eq", "Name" average around 4 characters.
	// Used for pre-allocating token slice capacity.
	estimatedAvgTokenLength = 4

	// minTokenSliceCapacity is the minimum capacity for the pre-allocated token slice.
	// Ensures small inputs don't result in tiny allocations that need immediate growth.
	minTokenSliceCapacity = 8

	// stringBuilderExtraCapacity is additional capacity added to strings.Builder
	// when handling escape sequences, to account for potential growth.
	stringBuilderExtraCapacity = 16
)

// TokenizeAll returns all tokens from the input
func (t *Tokenizer) TokenizeAll() ([]*Token, error) {
	// Pre-allocate token slice with estimated capacity based on input length
	estimatedTokens := len(t.input)/estimatedAvgTokenLength + 1
	if estimatedTokens < minTokenSliceCapacity {
		estimatedTokens = minTokenSliceCapacity
	}
	tokens := make([]*Token, 0, estimatedTokens)

	for {
		token, err := t.NextToken()
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, token)

		if token.Type == TokenEOF {
			break
		}
	}

	return tokens, nil
}
