package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseFilter parses a filter expression with metadata validation.
// It keeps 4.01-compatible behavior by default for backwards compatibility with existing callers.
func parseFilter(filterStr string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool, maxInClauseSize int) (*FilterExpression, error) {
	return parseFilterWithMode(filterStr, entityMetadata, computedAliases, maxInClauseSize, true)
}

func parseFilterWithMode(filterStr string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool, maxInClauseSize int, caseInsensitive bool) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Use pooled tokenizer and AST parser
	tokenizer := AcquireTokenizerWithMode(filterStr, caseInsensitive)
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		ReleaseTokenizer(tokenizer)
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}
	ReleaseTokenizer(tokenizer)

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	// Release AST nodes after conversion to reduce memory usage.
	// The FilterExpression holds FilterExpression references (not AST nodes) for nested functions.
	defer ReleaseASTNode(ast)

	return ASTToFilterExpressionWithComputedAndMode(ast, entityMetadata, computedAliases, maxInClauseSize, caseInsensitive)
}

// ParseFilterWithoutMetadata parses a filter expression without metadata validation
func ParseFilterWithoutMetadata(filterStr string) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Use pooled tokenizer and AST parser
	tokenizer := AcquireTokenizerWithMode(filterStr, true)
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		ReleaseTokenizer(tokenizer)
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}
	ReleaseTokenizer(tokenizer)

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	// Release AST nodes after conversion to reduce memory usage.
	// The FilterExpression holds FilterExpression references (not AST nodes) for nested functions.
	defer ReleaseASTNode(ast)

	// Convert AST to FilterExpression without metadata validation
	return ASTToFilterExpressionWithMode(ast, nil, true)
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
