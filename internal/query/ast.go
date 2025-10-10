package query

// ASTNode represents a node in the abstract syntax tree
type ASTNode interface {
	astNode()
}

// BinaryExpr represents a binary expression (e.g., A and B, X + Y)
type BinaryExpr struct {
	Left     ASTNode
	Operator string
	Right    ASTNode
}

func (e *BinaryExpr) astNode() {}

// UnaryExpr represents a unary expression (e.g., not X)
type UnaryExpr struct {
	Operator string
	Operand  ASTNode
}

func (e *UnaryExpr) astNode() {}

// ComparisonExpr represents a comparison (e.g., Price gt 100)
type ComparisonExpr struct {
	Left     ASTNode
	Operator string
	Right    ASTNode
}

func (e *ComparisonExpr) astNode() {}

// FunctionCallExpr represents a function call (e.g., contains(Name, 'text'))
type FunctionCallExpr struct {
	Function string
	Args     []ASTNode
}

func (e *FunctionCallExpr) astNode() {}

// IdentifierExpr represents an identifier (property name)
type IdentifierExpr struct {
	Name string
}

func (e *IdentifierExpr) astNode() {}

// LiteralExpr represents a literal value
type LiteralExpr struct {
	Value interface{}
	Type  string // "string", "number", "boolean", "null"
}

func (e *LiteralExpr) astNode() {}

// GroupExpr represents a grouped expression (parentheses)
type GroupExpr struct {
	Expr ASTNode
}

func (e *GroupExpr) astNode() {}
