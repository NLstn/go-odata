package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// ParseSearchExpression — parser correctness
// ---------------------------------------------------------------------------

func TestParseSearchExpression_SingleTerm(t *testing.T) {
	node := ParseSearchExpression("laptop")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op != searchOpTerm {
		t.Errorf("expected searchOpTerm, got %v", node.op)
	}
	if node.term != "laptop" {
		t.Errorf("expected term 'laptop', got %q", node.term)
	}
}

func TestParseSearchExpression_EmptyQuery(t *testing.T) {
	if node := ParseSearchExpression(""); node != nil {
		t.Errorf("expected nil for empty query, got %+v", node)
	}
	if node := ParseSearchExpression("   "); node != nil {
		t.Errorf("expected nil for whitespace-only query, got %+v", node)
	}
}

func TestParseSearchExpression_QuotedPhrase(t *testing.T) {
	node := ParseSearchExpression(`"high performance"`)
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op != searchOpPhrase {
		t.Errorf("expected searchOpPhrase, got %v", node.op)
	}
	if node.term != "high performance" {
		t.Errorf("expected 'high performance', got %q", node.term)
	}
}

func TestParseSearchExpression_ExplicitAND(t *testing.T) {
	node := ParseSearchExpression("laptop AND wireless")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op != searchOpAnd {
		t.Errorf("expected searchOpAnd, got %v", node.op)
	}
	if node.left == nil || node.left.term != "laptop" {
		t.Error("expected left term 'laptop'")
	}
	if node.right == nil || node.right.term != "wireless" {
		t.Error("expected right term 'wireless'")
	}
}

func TestParseSearchExpression_ImplicitAND(t *testing.T) {
	// Adjacent terms with no keyword produce an implicit AND node
	node := ParseSearchExpression("laptop wireless")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op != searchOpAnd {
		t.Errorf("expected implicit AND (searchOpAnd), got %v", node.op)
	}
}

func TestParseSearchExpression_OR(t *testing.T) {
	node := ParseSearchExpression("laptop OR phone")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op != searchOpOr {
		t.Errorf("expected searchOpOr, got %v", node.op)
	}
	if node.left == nil || node.left.term != "laptop" {
		t.Error("expected left term 'laptop'")
	}
	if node.right == nil || node.right.term != "phone" {
		t.Error("expected right term 'phone'")
	}
}

func TestParseSearchExpression_NOT(t *testing.T) {
	node := ParseSearchExpression("NOT phone")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op != searchOpNot {
		t.Errorf("expected searchOpNot, got %v", node.op)
	}
	if node.left == nil || node.left.term != "phone" {
		t.Error("expected operand 'phone'")
	}
}

func TestParseSearchExpression_GroupedOR(t *testing.T) {
	node := ParseSearchExpression("(laptop OR phone) AND wireless")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op != searchOpAnd {
		t.Errorf("expected top-level AND, got %v", node.op)
	}
	if node.left == nil || node.left.op != searchOpOr {
		t.Error("expected left child to be OR")
	}
	if node.right == nil || node.right.term != "wireless" {
		t.Error("expected right child to be 'wireless'")
	}
}

func TestParseSearchExpression_OperatorPrecedence(t *testing.T) {
	// "a OR b AND c" → "a OR (b AND c)" (AND > OR)
	node := ParseSearchExpression("a OR b AND c")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op != searchOpOr {
		t.Fatalf("expected top-level OR, got %v", node.op)
	}
	if node.right == nil || node.right.op != searchOpAnd {
		t.Errorf("expected OR right to be AND, got %v", node.right)
	}
}

func TestParseSearchExpression_LowercaseKeywordsArePlainTerms(t *testing.T) {
	// Lowercase "or" must be treated as a plain search term, not an OR operator
	node := ParseSearchExpression("laptop or phone")
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.op == searchOpOr {
		t.Error("lowercase 'or' should not be treated as an OR operator")
	}
}

// ---------------------------------------------------------------------------
// FTS query serialization
// ---------------------------------------------------------------------------

func TestToFTS5Query(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"laptop", "laptop"},
		{`"high performance"`, `"high performance"`},
		{"laptop AND wireless", "laptop AND wireless"},
		{"laptop OR phone", "(laptop OR phone)"},
		{"NOT phone", "NOT phone"},
		{"(laptop OR phone) AND wireless", "(laptop OR phone) AND wireless"},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			node := ParseSearchExpression(tt.query)
			if node == nil {
				t.Fatalf("ParseSearchExpression returned nil for %q", tt.query)
			}
			got := node.toFTS5Query()
			if got != tt.expected {
				t.Errorf("toFTS5Query(%q) = %q, want %q", tt.query, got, tt.expected)
			}
		})
	}
}

func TestToFTS34Query(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"laptop", "laptop"},
		{"laptop AND wireless", "laptop wireless"},
		{"laptop OR phone", "(laptop OR phone)"},
		// NOT is not supported by FTS3/4 and is dropped
		{"laptop NOT phone", "laptop"},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			node := ParseSearchExpression(tt.query)
			if node == nil {
				t.Fatalf("ParseSearchExpression returned nil for %q", tt.query)
			}
			got := node.toFTS34Query()
			if got != tt.expected {
				t.Errorf("toFTS34Query(%q) = %q, want %q", tt.query, got, tt.expected)
			}
		})
	}
}

func TestToWebsearchQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"laptop", "laptop"},
		{`"high performance"`, `"high performance"`},
		{"laptop AND wireless", "laptop wireless"},
		{"laptop OR phone", "laptop or phone"},
		{"NOT phone", "-phone"},
		{`NOT "high end"`, `-"high end"`},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			node := ParseSearchExpression(tt.query)
			if node == nil {
				t.Fatalf("ParseSearchExpression returned nil for %q", tt.query)
			}
			got := node.toWebsearchQuery()
			if got != tt.expected {
				t.Errorf("toWebsearchQuery(%q) = %q, want %q", tt.query, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// In-memory boolean operator evaluation (issues 1, 3, 4)
// ---------------------------------------------------------------------------

// SearchTestEntityMixed has both string and non-string searchable fields (issue 4)
type SearchTestEntityMixed struct {
	ID       int     `json:"ID" odata:"key"`
	Name     string  `json:"Name" odata:"searchable"`
	Price    float64 `json:"Price" odata:"searchable"`
	InStock  bool    `json:"InStock" odata:"searchable"`
	Quantity int     `json:"Quantity" odata:"searchable"`
}

func TestApplySearch_OR(t *testing.T) {
	// Issue 1: OR operator must return results matching either term
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity: %v", err)
	}

	entities := []SearchTestEntity{
		{ID: 1, Name: "Laptop Pro", Description: "laptop device"},
		{ID: 2, Name: "Smartphone", Description: "mobile phone device"},
		{ID: 3, Name: "Coffee Mug", Description: "kitchen item"},
	}

	result := ApplySearch(entities, "laptop OR phone", meta)
	resultSlice, ok := result.([]SearchTestEntity)
	if !ok {
		t.Fatalf("unexpected type %T", result)
	}
	if len(resultSlice) != 2 {
		t.Errorf("OR search: expected 2 results, got %d", len(resultSlice))
	}
	ids := make(map[int]bool)
	for _, e := range resultSlice {
		ids[e.ID] = true
	}
	if !ids[1] {
		t.Error("OR search: entity 1 (Laptop) should match")
	}
	if !ids[2] {
		t.Error("OR search: entity 2 (Smartphone/phone) should match")
	}
	if ids[3] {
		t.Error("OR search: entity 3 (Coffee Mug) should not match")
	}
}

func TestApplySearch_AND_BooleanOperator(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity: %v", err)
	}

	entities := []SearchTestEntity{
		{ID: 1, Name: "Laptop Pro", Description: "gaming laptop"},
		{ID: 2, Name: "Gaming Mouse", Description: "gaming device"},
		{ID: 3, Name: "Coffee Mug", Description: "kitchen item"},
	}

	result := ApplySearch(entities, "gaming AND laptop", meta)
	resultSlice, ok := result.([]SearchTestEntity)
	if !ok {
		t.Fatalf("unexpected type %T", result)
	}
	if len(resultSlice) != 1 {
		t.Errorf("AND search: expected 1 result, got %d", len(resultSlice))
	}
	if len(resultSlice) > 0 && resultSlice[0].ID != 1 {
		t.Errorf("AND search: expected entity 1, got %d", resultSlice[0].ID)
	}
}

func TestApplySearch_NOT_BooleanOperator(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity: %v", err)
	}

	entities := []SearchTestEntity{
		{ID: 1, Name: "Laptop Pro", Description: "gaming laptop"},
		{ID: 2, Name: "Gaming Mouse", Description: "gaming device"},
		{ID: 3, Name: "Coffee Mug", Description: "kitchen item"},
	}

	// "gaming NOT laptop" → entity 2 matches, entity 1 has both gaming+laptop so NOT laptop excludes it
	result := ApplySearch(entities, "gaming NOT laptop", meta)
	resultSlice, ok := result.([]SearchTestEntity)
	if !ok {
		t.Fatalf("unexpected type %T", result)
	}
	if len(resultSlice) != 1 {
		t.Errorf("NOT search: expected 1 result, got %d", len(resultSlice))
	}
	if len(resultSlice) > 0 && resultSlice[0].ID != 2 {
		t.Errorf("NOT search: expected entity 2 (Gaming Mouse), got %d", resultSlice[0].ID)
	}
}

func TestApplySearch_PhraseSearch(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity: %v", err)
	}

	entities := []SearchTestEntity{
		{ID: 1, Name: "High Performance Laptop", Description: "power device"},
		{ID: 2, Name: "Laptop Pro", Description: "high end laptop"},
		{ID: 3, Name: "Coffee Mug", Description: "kitchen item"},
	}

	result := ApplySearch(entities, `"high performance"`, meta)
	resultSlice, ok := result.([]SearchTestEntity)
	if !ok {
		t.Fatalf("unexpected type %T", result)
	}
	if len(resultSlice) != 1 {
		t.Errorf("phrase search: expected 1 result, got %d", len(resultSlice))
	}
	if len(resultSlice) > 0 && resultSlice[0].ID != 1 {
		t.Errorf("phrase search: expected entity 1, got %d", resultSlice[0].ID)
	}
}

// Issue 4: non-string fields tagged searchable must participate in search
func TestApplySearch_NonStringFields(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntityMixed{})
	if err != nil {
		t.Fatalf("AnalyzeEntity: %v", err)
	}

	entities := []SearchTestEntityMixed{
		{ID: 1, Name: "Widget", Price: 42.99, InStock: true, Quantity: 100},
		{ID: 2, Name: "Gadget", Price: 9.99, InStock: false, Quantity: 50},
	}

	// Search by float value
	result := ApplySearch(entities, "42.99", meta)
	s, ok := result.([]SearchTestEntityMixed)
	if !ok {
		t.Fatalf("unexpected type %T", result)
	}
	if len(s) != 1 || s[0].ID != 1 {
		t.Errorf("float search: expected entity 1, got %v", s)
	}

	// Search by bool value
	result = ApplySearch(entities, "true", meta)
	s, ok = result.([]SearchTestEntityMixed)
	if !ok {
		t.Fatalf("unexpected type %T", result)
	}
	if len(s) != 1 || s[0].ID != 1 {
		t.Errorf("bool search: expected entity 1, got %v", s)
	}

	// Search by int value
	result = ApplySearch(entities, "100", meta)
	s, ok = result.([]SearchTestEntityMixed)
	if !ok {
		t.Fatalf("unexpected type %T", result)
	}
	if len(s) != 1 || s[0].ID != 1 {
		t.Errorf("int search: expected entity 1, got %v", s)
	}
}

// Issue 3: fuzzy matching must be rune-aware for multi-byte UTF-8 characters
func TestFuzzyContains_UTF8(t *testing.T) {
	tests := []struct {
		text     string
		pattern  string
		fuzz     int
		expected bool
	}{
		// ASCII sanity checks
		{"hello world", "world", 1, true},
		{"hello world", "xyz", 1, false},
		// Accented characters (multi-byte in UTF-8)
		{"café latte", "café", 1, true},
		{"café latte", "cafe", 2, true}, // 1 char diff (é vs e)
		// CJK characters (each is 3 bytes in UTF-8)
		{"東京都", "東京", 1, true},
		{"東京都", "大阪", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.text+"_"+tt.pattern, func(t *testing.T) {
			got := fuzzyContains(tt.text, tt.pattern, tt.fuzz)
			if got != tt.expected {
				t.Errorf("fuzzyContains(%q, %q, %d) = %v, want %v", tt.text, tt.pattern, tt.fuzz, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SQLite FTS boolean operators (issue 1 via FTS)
// ---------------------------------------------------------------------------

func newFTSTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestFTSManager_ApplyFTSSearch_OR(t *testing.T) {
	db := newFTSTestDB(t)
	if err := db.AutoMigrate(&FTSTestEntity{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	testData := []FTSTestEntity{
		{ID: 1, Name: "Laptop Pro", Description: "laptop device", Category: "Electronics"},
		{ID: 2, Name: "Smartphone", Description: "mobile phone", Category: "Electronics"},
		{ID: 3, Name: "Coffee Mug", Description: "kitchen item", Category: "Kitchen"},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	mgr := NewFTSManager(db)
	if !mgr.IsFTSAvailable() {
		t.Skip("FTS not available")
	}

	meta, err := metadata.AnalyzeEntity(FTSTestEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity: %v", err)
	}

	q := db.Table("fts_test_entities")
	q, err = mgr.ApplyFTSSearch(q, "fts_test_entities", "laptop OR phone", meta)
	if err != nil {
		t.Fatalf("ApplyFTSSearch: %v", err)
	}

	var results []FTSTestEntity
	if err := q.Find(&results).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("FTS OR: expected 2 results, got %d", len(results))
	}
}

func TestFTSManager_ApplyFTSSearch_NOT(t *testing.T) {
	db := newFTSTestDB(t)
	if err := db.AutoMigrate(&FTSTestEntity{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	testData := []FTSTestEntity{
		{ID: 1, Name: "Laptop Pro", Description: "gaming laptop", Category: "Electronics"},
		{ID: 2, Name: "Gaming Mouse", Description: "gaming device", Category: "Electronics"},
		{ID: 3, Name: "Coffee Mug", Description: "kitchen item", Category: "Kitchen"},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	mgr := NewFTSManager(db)
	if !mgr.IsFTSAvailable() {
		t.Skip("FTS not available")
	}
	if mgr.GetFTSVersion() != "FTS5" {
		t.Skip("NOT operator requires FTS5; skipping on older FTS versions")
	}

	meta, err := metadata.AnalyzeEntity(FTSTestEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity: %v", err)
	}

	q := db.Table("fts_test_entities")
	q, err = mgr.ApplyFTSSearch(q, "fts_test_entities", "gaming NOT laptop", meta)
	if err != nil {
		t.Fatalf("ApplyFTSSearch: %v", err)
	}

	var results []FTSTestEntity
	if err := q.Find(&results).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("FTS NOT: expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != 2 {
		t.Errorf("FTS NOT: expected entity 2, got %d", results[0].ID)
	}
}

// Issue 6: FTS language must be configurable
func TestFTSManager_WithLanguageOption(t *testing.T) {
	db := newFTSTestDB(t)
	mgr := NewFTSManagerWithOptions(db, FTSOptions{Language: "simple"})
	if mgr == nil {
		t.Fatal("expected non-nil FTSManager")
	}
	if mgr.language != "simple" {
		t.Errorf("expected language 'simple', got %q", mgr.language)
	}
}

func TestFTSManager_DefaultLanguageIsEnglish(t *testing.T) {
	db := newFTSTestDB(t)
	mgr := NewFTSManager(db)
	if mgr.language != "english" {
		t.Errorf("expected default language 'english', got %q", mgr.language)
	}
}
