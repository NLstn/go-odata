package metadata

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// AnnotationTarget represents the type of element that can be annotated
type AnnotationTarget string

const (
	// TargetEntityType represents an EntityType target for annotations
	TargetEntityType AnnotationTarget = "EntityType"
	// TargetProperty represents a Property target for annotations
	TargetProperty AnnotationTarget = "Property"
	// TargetNavigationProperty represents a NavigationProperty target for annotations
	TargetNavigationProperty AnnotationTarget = "NavigationProperty"
	// TargetEntitySet represents an EntitySet target for annotations
	TargetEntitySet AnnotationTarget = "EntitySet"
	// TargetSingleton represents a Singleton target for annotations
	TargetSingleton AnnotationTarget = "Singleton"
	// TargetEntityContainer represents an EntityContainer target for annotations
	TargetEntityContainer AnnotationTarget = "EntityContainer"
)

// Vocabulary represents an OData vocabulary with a namespace and alias
type Vocabulary struct {
	Namespace string
	Alias     string
}

// Common OData vocabularies
var (
	// CoreVocabulary is the OData Core vocabulary (Org.OData.Core.V1)
	CoreVocabulary = Vocabulary{
		Namespace: "Org.OData.Core.V1",
		Alias:     "Core",
	}

	// CapabilitiesVocabulary is the OData Capabilities vocabulary (Org.OData.Capabilities.V1)
	CapabilitiesVocabulary = Vocabulary{
		Namespace: "Org.OData.Capabilities.V1",
		Alias:     "Capabilities",
	}

	// ValidationVocabulary is the OData Validation vocabulary (Org.OData.Validation.V1)
	ValidationVocabulary = Vocabulary{
		Namespace: "Org.OData.Validation.V1",
		Alias:     "Validation",
	}
)

// Annotation represents an OData annotation
type Annotation struct {
	// Term is the fully qualified name of the annotation term (e.g., "Org.OData.Core.V1.Computed")
	Term string
	// Value is the annotation value. Can be a primitive, record, or collection.
	Value interface{}
	// Qualifier is an optional qualifier for the annotation
	Qualifier string
}

// QualifiedTerm returns the term combined with its qualifier if present (e.g., "Org.OData.Core.V1.Display#Short")
func (a *Annotation) QualifiedTerm() string {
	if a == nil || a.Term == "" {
		return ""
	}
	if a.Qualifier != "" {
		return a.Term + "#" + a.Qualifier
	}
	return a.Term
}

// TermName returns just the term name without the namespace (e.g., "Computed" from "Org.OData.Core.V1.Computed")
func (a *Annotation) TermName() string {
	if a == nil || a.Term == "" {
		return ""
	}
	parts := strings.Split(a.Term, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return a.Term
}

// VocabularyNamespace returns the vocabulary namespace (e.g., "Org.OData.Core.V1" from "Org.OData.Core.V1.Computed")
func (a *Annotation) VocabularyNamespace() string {
	if a == nil || a.Term == "" {
		return ""
	}
	lastDot := strings.LastIndex(a.Term, ".")
	if lastDot > 0 {
		return a.Term[:lastDot]
	}
	return ""
}

// AnnotationCollection holds a collection of annotations for a target
type AnnotationCollection struct {
	annotations []Annotation
}

// NewAnnotationCollection creates a new empty annotation collection
func NewAnnotationCollection() *AnnotationCollection {
	return &AnnotationCollection{
		annotations: make([]Annotation, 0),
	}
}

// Add adds an annotation to the collection
func (c *AnnotationCollection) Add(annotation Annotation) {
	if c == nil {
		return
	}
	c.annotations = append(c.annotations, annotation)
}

// AddTerm adds an annotation with the given term and value
func (c *AnnotationCollection) AddTerm(term string, value interface{}) {
	if c == nil {
		return
	}
	c.annotations = append(c.annotations, Annotation{
		Term:  term,
		Value: value,
	})
}

// Get returns all annotations in the collection
func (c *AnnotationCollection) Get() []Annotation {
	if c == nil {
		return nil
	}
	return c.annotations
}

// GetByTerm returns all annotations with the specified term
func (c *AnnotationCollection) GetByTerm(term string) []Annotation {
	if c == nil {
		return nil
	}
	result := make([]Annotation, 0)
	for _, a := range c.annotations {
		if a.Term == term {
			result = append(result, a)
		}
	}
	return result
}

// GetByVocabulary returns all annotations from the specified vocabulary namespace
func (c *AnnotationCollection) GetByVocabulary(namespace string) []Annotation {
	if c == nil {
		return nil
	}
	result := make([]Annotation, 0)
	for _, a := range c.annotations {
		if a.VocabularyNamespace() == namespace {
			result = append(result, a)
		}
	}
	return result
}

// Has returns true if an annotation with the given term exists
func (c *AnnotationCollection) Has(term string) bool {
	if c == nil {
		return false
	}
	for _, a := range c.annotations {
		if a.Term == term {
			return true
		}
	}
	return false
}

// Len returns the number of annotations in the collection
func (c *AnnotationCollection) Len() int {
	if c == nil {
		return 0
	}
	return len(c.annotations)
}

// UsedVocabularies returns a list of unique vocabulary namespaces used in the annotations
func (c *AnnotationCollection) UsedVocabularies() []string {
	if c == nil {
		return nil
	}
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, a := range c.annotations {
		ns := a.VocabularyNamespace()
		if ns != "" && !seen[ns] {
			seen[ns] = true
			result = append(result, ns)
		}
	}
	return result
}

// Common OData Core vocabulary term constants
const (
	// CoreComputed indicates that the property value is computed by the service
	CoreComputed = "Org.OData.Core.V1.Computed"
	// CoreImmutable indicates that the property value cannot be changed after creation
	CoreImmutable = "Org.OData.Core.V1.Immutable"
	// CoreDescription provides a human-readable description
	CoreDescription = "Org.OData.Core.V1.Description"
	// CoreLongDescription provides a detailed human-readable description
	CoreLongDescription = "Org.OData.Core.V1.LongDescription"
	// CorePermissions indicates read/write permissions for a property
	CorePermissions = "Org.OData.Core.V1.Permissions"
	// CoreOptimisticConcurrency indicates properties used for ETag computation
	CoreOptimisticConcurrency = "Org.OData.Core.V1.OptimisticConcurrency"
)

// Common OData Capabilities vocabulary term constants
const (
	// CapInsertRestrictions indicates insert restrictions
	CapInsertRestrictions = "Org.OData.Capabilities.V1.InsertRestrictions"
	// CapUpdateRestrictions indicates update restrictions
	CapUpdateRestrictions = "Org.OData.Capabilities.V1.UpdateRestrictions"
	// CapDeleteRestrictions indicates delete restrictions
	CapDeleteRestrictions = "Org.OData.Capabilities.V1.DeleteRestrictions"
	// CapReadRestrictions indicates read restrictions
	CapReadRestrictions = "Org.OData.Capabilities.V1.ReadRestrictions"
	// CapFilterRestrictions indicates filter restrictions
	CapFilterRestrictions = "Org.OData.Capabilities.V1.FilterRestrictions"
	// CapSortRestrictions indicates sort restrictions
	CapSortRestrictions = "Org.OData.Capabilities.V1.SortRestrictions"
	// CapSearchRestrictions indicates search restrictions
	CapSearchRestrictions = "Org.OData.Capabilities.V1.SearchRestrictions"
	// CapCountRestrictions indicates count restrictions
	CapCountRestrictions = "Org.OData.Capabilities.V1.CountRestrictions"
	// CapExpandRestrictions indicates expand restrictions
	CapExpandRestrictions = "Org.OData.Capabilities.V1.ExpandRestrictions"
	// CapSelectSupport indicates select support
	CapSelectSupport = "Org.OData.Capabilities.V1.SelectSupport"
)

// ParseAnnotationTag parses an annotation tag value and returns the term and value.
// Tag format: "term=value" or just "term" for boolean true.
// Qualifiers can be specified as "term#Qualifier" or by appending ";qualifier=Qualifier".
// Examples:
//   - "Org.OData.Core.V1.Computed" -> term: "Org.OData.Core.V1.Computed", value: true
//   - "Core.Computed" -> term: "Org.OData.Core.V1.Computed", value: true (with alias expansion)
//   - "Org.OData.Core.V1.Description=Product name" -> term: "...", value: "Product name"
func ParseAnnotationTag(tag string) (Annotation, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return Annotation{}, fmt.Errorf("empty annotation tag")
	}

	segments := strings.Split(tag, ";")
	termValue := strings.TrimSpace(segments[0])
	var qualifier string

	for _, segment := range segments[1:] {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if strings.HasPrefix(segment, "qualifier=") {
			qualifierValue := strings.TrimSpace(strings.TrimPrefix(segment, "qualifier="))
			if qualifierValue == "" {
				return Annotation{}, fmt.Errorf("empty annotation qualifier")
			}
			if qualifier != "" && qualifier != qualifierValue {
				return Annotation{}, fmt.Errorf("conflicting annotation qualifiers")
			}
			qualifier = qualifierValue
			continue
		}
		return Annotation{}, fmt.Errorf("unsupported annotation tag segment: %q", segment)
	}

	// Check for term=value format
	parts := strings.SplitN(termValue, "=", 2)
	term := strings.TrimSpace(parts[0])

	if strings.Contains(term, "#") {
		termParts := strings.SplitN(term, "#", 2)
		term = strings.TrimSpace(termParts[0])
		hashQualifier := strings.TrimSpace(termParts[1])
		if hashQualifier == "" {
			return Annotation{}, fmt.Errorf("empty annotation qualifier")
		}
		if qualifier != "" && qualifier != hashQualifier {
			return Annotation{}, fmt.Errorf("conflicting annotation qualifiers")
		}
		qualifier = hashQualifier
	}

	// Validate that term is not empty after trimming and extracting qualifier
	if term == "" {
		return Annotation{}, fmt.Errorf("empty annotation term")
	}

	var value interface{} = true // Default to boolean true for bare terms

	if len(parts) == 2 {
		literal := strings.TrimSpace(parts[1])
		parsedValue, err := parseAnnotationLiteral(literal)
		if err != nil {
			return Annotation{}, err
		}
		value = parsedValue
	}

	// Expand common aliases
	term = expandAnnotationAlias(term)

	return Annotation{
		Term:      term,
		Value:     value,
		Qualifier: qualifier,
	}, nil
}

var (
	annotationIntLiteralPattern   = regexp.MustCompile(`^[+-]?\d+$`)
	annotationFloatLiteralPattern = regexp.MustCompile(`^[+-]?(?:\d+\.\d*|\d*\.\d+|\d+[eE][+-]?\d+|\d+\.\d*[eE][+-]?\d+|\d*\.\d+[eE][+-]?\d+)$`)
)

func parseAnnotationLiteral(literal string) (interface{}, error) {
	if literal == "" {
		return "", nil
	}

	if unquoted, ok, err := parseQuotedLiteral(literal); ok {
		return unquoted, err
	}

	if strings.EqualFold(literal, "true") {
		return true, nil
	}
	if strings.EqualFold(literal, "false") {
		return false, nil
	}

	if annotationIntLiteralPattern.MatchString(literal) {
		parsed, err := strconv.ParseInt(literal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer literal %q: %w", literal, err)
		}
		return parsed, nil
	}

	if annotationFloatLiteralPattern.MatchString(literal) {
		parsed, err := strconv.ParseFloat(literal, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float literal %q: %w", literal, err)
		}
		return parsed, nil
	}

	if looksNumericLiteral(literal) {
		return nil, fmt.Errorf("invalid numeric literal %q", literal)
	}

	return literal, nil
}

func parseQuotedLiteral(literal string) (string, bool, error) {
	if len(literal) < 2 {
		return "", false, nil
	}
	quote := literal[0]
	if quote != '"' && quote != '\'' {
		return "", false, nil
	}
	if literal[len(literal)-1] != quote {
		return "", true, fmt.Errorf("unterminated string literal %q", literal)
	}
	return literal[1 : len(literal)-1], true, nil
}

func looksNumericLiteral(literal string) bool {
	hasDigit := false
	for _, r := range literal {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '+' || r == '-' || r == '.' || r == 'e' || r == 'E':
			continue
		default:
			return false
		}
	}
	return hasDigit
}

// expandAnnotationAlias expands common vocabulary aliases to full namespaces
func expandAnnotationAlias(term string) string {
	// Handle Core.* -> Org.OData.Core.V1.*
	if strings.HasPrefix(term, "Core.") {
		return "Org.OData.Core.V1." + term[5:]
	}
	// Handle Capabilities.* -> Org.OData.Capabilities.V1.*
	if strings.HasPrefix(term, "Capabilities.") {
		return "Org.OData.Capabilities.V1." + term[13:]
	}
	// Handle Validation.* -> Org.OData.Validation.V1.*
	if strings.HasPrefix(term, "Validation.") {
		return "Org.OData.Validation.V1." + term[11:]
	}
	return term
}

// ExpandAnnotationAlias expands common vocabulary aliases to full namespaces.
func ExpandAnnotationAlias(term string) string {
	return expandAnnotationAlias(term)
}

// VocabularyAliasMap returns a map of vocabulary namespace to preferred alias
func VocabularyAliasMap() map[string]string {
	return map[string]string{
		"Org.OData.Core.V1":         "Core",
		"Org.OData.Capabilities.V1": "Capabilities",
		"Org.OData.Validation.V1":   "Validation",
	}
}
