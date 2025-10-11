package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseFilter parses a filter expression with metadata validation
func parseFilter(filterStr string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Use the tokenizer and AST parser
	tokenizer := NewTokenizer(filterStr)
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	return ASTToFilterExpression(ast, entityMetadata)
}

// parseFilterWithoutMetadata parses a filter expression without metadata validation
func parseFilterWithoutMetadata(filterStr string) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Use the AST parser
	tokenizer := NewTokenizer(filterStr)
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	// Convert AST to FilterExpression without metadata validation
	return ASTToFilterExpression(ast, nil)
}

// splitFunctionArgs splits function arguments by comma
func splitFunctionArgs(args string) []string {
	result := make([]string, 0)
	current := ""
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range args {
		if ch == '\'' || ch == '"' {
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
				quoteChar = 0
			}
			current += string(ch)
		} else if ch == ',' && !inQuotes {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
