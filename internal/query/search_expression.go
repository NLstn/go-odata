package query

import "strings"

// searchOp is the type of node in a parsed $search expression tree.
type searchOp int

const (
	searchOpTerm   searchOp = iota // Single word — must appear in the field
	searchOpPhrase                 // Quoted phrase — must appear as a consecutive substring
	searchOpAnd                    // Both sub-expressions must match
	searchOpOr                     // At least one sub-expression must match
	searchOpNot                    // Sub-expression must NOT match
)

// SearchExprNode is a node in the parsed OData $search expression tree.
// The grammar follows OData v4 §11.2.5.6:
//
//	searchExpr  = orExpr
//	orExpr      = andExpr ("OR" andExpr)*
//	andExpr     = notExpr ("AND" notExpr | notExpr)*   // implicit AND between adjacent terms
//	notExpr     = "NOT" notExpr | primary
//	primary     = DQUOTE phrase DQUOTE | "(" orExpr ")" | term
type SearchExprNode struct {
	op    searchOp
	term  string          // set for searchOpTerm and searchOpPhrase
	left  *SearchExprNode // set for searchOpAnd, searchOpOr, searchOpNot
	right *SearchExprNode // set for searchOpAnd and searchOpOr; nil for searchOpNot
}

// ParseSearchExpression parses an OData $search query string into an expression tree.
// The parser is lenient: malformed expressions degrade to simple term matching rather
// than returning an error, so a bad $search value never produces a 500.
func ParseSearchExpression(query string) *SearchExprNode {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	p := &searchParser{tokens: tokenizeSearch(query)}
	node := p.parseOr()
	return node
}

// --- tokenizer ---------------------------------------------------------------

type searchTokType int

const (
	sTokEOF    searchTokType = iota
	sTokTerm                 // any word that is not a keyword
	sTokPhrase               // "quoted phrase"
	sTokAND
	sTokOR
	sTokNOT
	sTokLParen
	sTokRParen
)

type searchTok struct {
	typ  searchTokType
	text string
}

func tokenizeSearch(input string) []searchTok {
	runes := []rune(input)
	n := len(runes)
	var toks []searchTok
	i := 0

	for i < n {
		// skip spaces
		for i < n && runes[i] == ' ' {
			i++
		}
		if i >= n {
			break
		}

		switch runes[i] {
		case '(':
			toks = append(toks, searchTok{typ: sTokLParen})
			i++
		case ')':
			toks = append(toks, searchTok{typ: sTokRParen})
			i++
		case '"':
			// quoted phrase — consume until closing quote
			i++ // skip opening "
			start := i
			for i < n && runes[i] != '"' {
				i++
			}
			phrase := string(runes[start:i])
			if i < n {
				i++ // skip closing "
			}
			toks = append(toks, searchTok{typ: sTokPhrase, text: phrase})
		default:
			start := i
			for i < n && runes[i] != ' ' && runes[i] != '(' && runes[i] != ')' && runes[i] != '"' {
				i++
			}
			word := string(runes[start:i])
			switch word {
			case "AND":
				toks = append(toks, searchTok{typ: sTokAND, text: word})
			case "OR":
				toks = append(toks, searchTok{typ: sTokOR, text: word})
			case "NOT":
				toks = append(toks, searchTok{typ: sTokNOT, text: word})
			default:
				toks = append(toks, searchTok{typ: sTokTerm, text: word})
			}
		}
	}

	toks = append(toks, searchTok{typ: sTokEOF})
	return toks
}

// --- recursive descent parser ------------------------------------------------

type searchParser struct {
	tokens []searchTok
	pos    int
}

func (p *searchParser) peek() searchTok {
	if p.pos >= len(p.tokens) {
		return searchTok{typ: sTokEOF}
	}
	return p.tokens[p.pos]
}

func (p *searchParser) consume() searchTok {
	tok := p.peek()
	if tok.typ != sTokEOF {
		p.pos++
	}
	return tok
}

// parseOr handles:  andExpr ("OR" andExpr)*
func (p *searchParser) parseOr() *SearchExprNode {
	left := p.parseAnd()
	if left == nil {
		return nil
	}
	for p.peek().typ == sTokOR {
		p.consume() // eat OR
		right := p.parseAnd()
		if right == nil {
			// Trailing OR with nothing on the right — treat OR as a term
			orTerm := &SearchExprNode{op: searchOpTerm, term: "OR"}
			left = &SearchExprNode{op: searchOpAnd, left: left, right: orTerm}
			break
		}
		left = &SearchExprNode{op: searchOpOr, left: left, right: right}
	}
	return left
}

// parseAnd handles:  notExpr ("AND" notExpr | notExpr)*
// Adjacent terms with no keyword between them are treated as implicit AND.
func (p *searchParser) parseAnd() *SearchExprNode {
	left := p.parseNot()
	if left == nil {
		return nil
	}
	for {
		tok := p.peek()
		if tok.typ == sTokAND {
			p.consume() // eat AND
			right := p.parseNot()
			if right == nil {
				// Trailing AND — treat AND as a term
				andTerm := &SearchExprNode{op: searchOpTerm, term: "AND"}
				left = &SearchExprNode{op: searchOpAnd, left: left, right: andTerm}
				break
			}
			left = &SearchExprNode{op: searchOpAnd, left: left, right: right}
		} else if tok.typ == sTokTerm || tok.typ == sTokPhrase || tok.typ == sTokNOT || tok.typ == sTokLParen {
			// Implicit AND: adjacent term/phrase/NOT/group
			right := p.parseNot()
			if right == nil {
				break
			}
			left = &SearchExprNode{op: searchOpAnd, left: left, right: right}
		} else {
			// OR or ) or EOF — stop
			break
		}
	}
	return left
}

// parseNot handles:  "NOT" notExpr | primary
func (p *searchParser) parseNot() *SearchExprNode {
	if p.peek().typ == sTokNOT {
		p.consume() // eat NOT
		operand := p.parseNot() // right-associative
		if operand == nil {
			// NOT with no operand — treat NOT as a term
			return &SearchExprNode{op: searchOpTerm, term: "NOT"}
		}
		return &SearchExprNode{op: searchOpNot, left: operand}
	}
	return p.parsePrimary()
}

// parsePrimary handles: DQUOTE phrase DQUOTE | "(" orExpr ")" | term
func (p *searchParser) parsePrimary() *SearchExprNode {
	tok := p.peek()
	switch tok.typ {
	case sTokPhrase:
		p.consume()
		return &SearchExprNode{op: searchOpPhrase, term: tok.text}
	case sTokTerm:
		p.consume()
		return &SearchExprNode{op: searchOpTerm, term: tok.text}
	case sTokLParen:
		p.consume() // eat '('
		inner := p.parseOr()
		if p.peek().typ == sTokRParen {
			p.consume() // eat ')'
		}
		return inner
	default:
		return nil
	}
}

// --- FTS query serialization --------------------------------------------------

// toFTS5Query converts the expression to SQLite FTS5 MATCH syntax.
// FTS5 supports AND, OR, NOT, and phrase search natively.
func (n *SearchExprNode) toFTS5Query() string {
	if n == nil {
		return ""
	}
	switch n.op {
	case searchOpTerm:
		return n.term
	case searchOpPhrase:
		return `"` + strings.ReplaceAll(n.term, `"`, `""`) + `"`
	case searchOpAnd:
		return n.left.toFTS5Query() + " AND " + n.right.toFTS5Query()
	case searchOpOr:
		return "(" + n.left.toFTS5Query() + " OR " + n.right.toFTS5Query() + ")"
	case searchOpNot:
		return "NOT " + wrapFTS5(n.left)
	}
	return ""
}

func wrapFTS5(n *SearchExprNode) string {
	if n == nil {
		return ""
	}
	if n.op == searchOpTerm || n.op == searchOpPhrase {
		return n.toFTS5Query()
	}
	return "(" + n.toFTS5Query() + ")"
}

// toFTS34Query converts the expression to SQLite FTS3/FTS4 MATCH syntax.
// FTS3/4 does not support NOT; NOT sub-expressions are silently dropped
// (the positive terms still participate in the AND/OR graph).
func (n *SearchExprNode) toFTS34Query() string {
	if n == nil {
		return ""
	}
	switch n.op {
	case searchOpTerm:
		return n.term
	case searchOpPhrase:
		return `"` + strings.ReplaceAll(n.term, `"`, `""`) + `"`
	case searchOpAnd:
		left := n.left.toFTS34Query()
		right := n.right.toFTS34Query()
		if left == "" {
			return right
		}
		if right == "" {
			return left
		}
		return left + " " + right
	case searchOpOr:
		return "(" + n.left.toFTS34Query() + " OR " + n.right.toFTS34Query() + ")"
	case searchOpNot:
		// NOT is not supported in FTS3/4 — drop the negated clause
		return ""
	}
	return ""
}

// toWebsearchQuery converts the expression to PostgreSQL websearch_to_tsquery syntax.
// Operator mapping:
//   - AND  → adjacent terms (implicit AND in websearch syntax)
//   - OR   → "or" keyword
//   - NOT  → "-" prefix (term and phrase only; complex sub-expressions use "-(…)")
//   - Term → bare word
//   - Phrase → "double-quoted"
func (n *SearchExprNode) toWebsearchQuery() string {
	if n == nil {
		return ""
	}
	switch n.op {
	case searchOpTerm:
		return n.term
	case searchOpPhrase:
		return `"` + strings.ReplaceAll(n.term, `"`, ``) + `"`
	case searchOpAnd:
		return n.left.toWebsearchQuery() + " " + n.right.toWebsearchQuery()
	case searchOpOr:
		return n.left.toWebsearchQuery() + " or " + n.right.toWebsearchQuery()
	case searchOpNot:
		inner := n.left
		switch inner.op {
		case searchOpTerm:
			return "-" + inner.term
		case searchOpPhrase:
			return `-"` + strings.ReplaceAll(inner.term, `"`, ``) + `"`
		default:
			// Complex NOT: wrap in parens; websearch_to_tsquery accepts -(…) syntax
			return "-(" + inner.toWebsearchQuery() + ")"
		}
	}
	return ""
}
