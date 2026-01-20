package metadata

import (
	"testing"
)

func TestAnnotation_TermName(t *testing.T) {
	tests := []struct {
		name     string
		term     string
		expected string
	}{
		{
			name:     "full Core vocabulary term",
			term:     "Org.OData.Core.V1.Computed",
			expected: "Computed",
		},
		{
			name:     "full Capabilities vocabulary term",
			term:     "Org.OData.Capabilities.V1.InsertRestrictions",
			expected: "InsertRestrictions",
		},
		{
			name:     "simple term",
			term:     "Computed",
			expected: "Computed",
		},
		{
			name:     "empty term",
			term:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation := &Annotation{Term: tt.term}
			got := annotation.TermName()
			if got != tt.expected {
				t.Errorf("TermName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAnnotation_VocabularyNamespace(t *testing.T) {
	tests := []struct {
		name     string
		term     string
		expected string
	}{
		{
			name:     "Core vocabulary",
			term:     "Org.OData.Core.V1.Computed",
			expected: "Org.OData.Core.V1",
		},
		{
			name:     "Capabilities vocabulary",
			term:     "Org.OData.Capabilities.V1.InsertRestrictions",
			expected: "Org.OData.Capabilities.V1",
		},
		{
			name:     "simple term without namespace",
			term:     "Computed",
			expected: "",
		},
		{
			name:     "empty term",
			term:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation := &Annotation{Term: tt.term}
			got := annotation.VocabularyNamespace()
			if got != tt.expected {
				t.Errorf("VocabularyNamespace() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAnnotation_NilAnnotation(t *testing.T) {
	var annotation *Annotation

	// Should not panic
	if got := annotation.TermName(); got != "" {
		t.Errorf("TermName() on nil = %v, want empty", got)
	}
	if got := annotation.VocabularyNamespace(); got != "" {
		t.Errorf("VocabularyNamespace() on nil = %v, want empty", got)
	}
}

func TestAnnotationCollection_Add(t *testing.T) {
	collection := NewAnnotationCollection()

	// Add some annotations
	collection.Add(Annotation{Term: CoreComputed, Value: true})
	collection.Add(Annotation{Term: CoreDescription, Value: "Test description"})

	if collection.Len() != 2 {
		t.Errorf("Len() = %d, want 2", collection.Len())
	}

	annotations := collection.Get()
	if len(annotations) != 2 {
		t.Errorf("Get() returned %d annotations, want 2", len(annotations))
	}
}

func TestAnnotationCollection_AddTerm(t *testing.T) {
	collection := NewAnnotationCollection()

	collection.AddTerm(CoreComputed, true)
	collection.AddTerm(CoreDescription, "Test description")

	if collection.Len() != 2 {
		t.Errorf("Len() = %d, want 2", collection.Len())
	}
}

func TestAnnotationCollection_GetByTerm(t *testing.T) {
	collection := NewAnnotationCollection()
	collection.AddTerm(CoreComputed, true)
	collection.AddTerm(CoreDescription, "Test description")
	collection.AddTerm(CoreComputed, false) // Duplicate term

	// Should find two annotations with CoreComputed
	computed := collection.GetByTerm(CoreComputed)
	if len(computed) != 2 {
		t.Errorf("GetByTerm(CoreComputed) returned %d, want 2", len(computed))
	}

	// Should find one annotation with CoreDescription
	descriptions := collection.GetByTerm(CoreDescription)
	if len(descriptions) != 1 {
		t.Errorf("GetByTerm(CoreDescription) returned %d, want 1", len(descriptions))
	}

	// Should find no annotations with unknown term
	unknown := collection.GetByTerm("Unknown.Term")
	if len(unknown) != 0 {
		t.Errorf("GetByTerm(Unknown) returned %d, want 0", len(unknown))
	}
}

func TestAnnotationCollection_GetByVocabulary(t *testing.T) {
	collection := NewAnnotationCollection()
	collection.AddTerm(CoreComputed, true)
	collection.AddTerm(CoreDescription, "Test description")
	collection.AddTerm(CapInsertRestrictions, map[string]interface{}{"Insertable": true})

	// Should find two annotations from Core vocabulary
	coreAnnotations := collection.GetByVocabulary("Org.OData.Core.V1")
	if len(coreAnnotations) != 2 {
		t.Errorf("GetByVocabulary(Core) returned %d, want 2", len(coreAnnotations))
	}

	// Should find one annotation from Capabilities vocabulary
	capAnnotations := collection.GetByVocabulary("Org.OData.Capabilities.V1")
	if len(capAnnotations) != 1 {
		t.Errorf("GetByVocabulary(Capabilities) returned %d, want 1", len(capAnnotations))
	}
}

func TestAnnotationCollection_Has(t *testing.T) {
	collection := NewAnnotationCollection()
	collection.AddTerm(CoreComputed, true)

	if !collection.Has(CoreComputed) {
		t.Error("Has(CoreComputed) = false, want true")
	}

	if collection.Has(CoreDescription) {
		t.Error("Has(CoreDescription) = true, want false")
	}
}

func TestAnnotationCollection_UsedVocabularies(t *testing.T) {
	collection := NewAnnotationCollection()
	collection.AddTerm(CoreComputed, true)
	collection.AddTerm(CoreDescription, "Test")
	collection.AddTerm(CapInsertRestrictions, map[string]interface{}{})

	vocabularies := collection.UsedVocabularies()
	if len(vocabularies) != 2 {
		t.Errorf("UsedVocabularies() returned %d, want 2", len(vocabularies))
	}

	// Check that both namespaces are present
	foundCore := false
	foundCap := false
	for _, ns := range vocabularies {
		if ns == "Org.OData.Core.V1" {
			foundCore = true
		}
		if ns == "Org.OData.Capabilities.V1" {
			foundCap = true
		}
	}

	if !foundCore {
		t.Error("UsedVocabularies() missing Core vocabulary")
	}
	if !foundCap {
		t.Error("UsedVocabularies() missing Capabilities vocabulary")
	}
}

func TestAnnotationCollection_NilCollection(t *testing.T) {
	var collection *AnnotationCollection

	// Should not panic
	collection.Add(Annotation{Term: CoreComputed, Value: true})
	collection.AddTerm(CoreComputed, true)

	if collection.Len() != 0 {
		t.Errorf("Len() on nil = %d, want 0", collection.Len())
	}
	if collection.Has(CoreComputed) {
		t.Error("Has() on nil = true, want false")
	}
	if collection.Get() != nil {
		t.Error("Get() on nil should return nil")
	}
	if collection.GetByTerm(CoreComputed) != nil {
		t.Error("GetByTerm() on nil should return nil")
	}
	if collection.GetByVocabulary("any") != nil {
		t.Error("GetByVocabulary() on nil should return nil")
	}
	if collection.UsedVocabularies() != nil {
		t.Error("UsedVocabularies() on nil should return nil")
	}
}

func TestParseAnnotationTag(t *testing.T) {
	tests := []struct {
		name              string
		tag               string
		expectedTerm      string
		expectedQualifier string
		expectedValue     interface{}
		expectError       bool
	}{
		{
			name:              "full term without value",
			tag:               "Org.OData.Core.V1.Computed",
			expectedTerm:      "Org.OData.Core.V1.Computed",
			expectedQualifier: "",
			expectedValue:     true,
		},
		{
			name:              "Core alias without value",
			tag:               "Core.Computed",
			expectedTerm:      "Org.OData.Core.V1.Computed",
			expectedQualifier: "",
			expectedValue:     true,
		},
		{
			name:              "Capabilities alias without value",
			tag:               "Capabilities.InsertRestrictions",
			expectedTerm:      "Org.OData.Capabilities.V1.InsertRestrictions",
			expectedQualifier: "",
			expectedValue:     true,
		},
		{
			name:              "term with string value",
			tag:               "Org.OData.Core.V1.Description=Product name",
			expectedTerm:      "Org.OData.Core.V1.Description",
			expectedQualifier: "",
			expectedValue:     "Product name",
		},
		{
			name:              "term with boolean true value",
			tag:               "Core.Computed=true",
			expectedTerm:      "Org.OData.Core.V1.Computed",
			expectedQualifier: "",
			expectedValue:     true,
		},
		{
			name:              "term with boolean false value",
			tag:               "Core.Computed=false",
			expectedTerm:      "Org.OData.Core.V1.Computed",
			expectedQualifier: "",
			expectedValue:     false,
		},
		{
			name:              "term with integer value",
			tag:               "Core.Permissions=42",
			expectedTerm:      "Org.OData.Core.V1.Permissions",
			expectedQualifier: "",
			expectedValue:     int64(42),
		},
		{
			name:              "term with float value",
			tag:               "Core.Permissions=3.14",
			expectedTerm:      "Org.OData.Core.V1.Permissions",
			expectedQualifier: "",
			expectedValue:     3.14,
		},
		{
			name:              "term with exponent float value",
			tag:               "Core.Permissions=1e3",
			expectedTerm:      "Org.OData.Core.V1.Permissions",
			expectedQualifier: "",
			expectedValue:     1000.0,
		},
		{
			name:              "quoted string literal",
			tag:               "Core.Description=\"true\"",
			expectedTerm:      "Org.OData.Core.V1.Description",
			expectedQualifier: "",
			expectedValue:     "true",
		},
		{
			name:        "invalid numeric literal",
			tag:         "Core.Permissions=1.2.3",
			expectError: true,
		},
		{
			name:        "unterminated quoted literal",
			tag:         "Core.Description=\"missing",
			expectError: true,
		},
		{
			name:              "alias with string value",
			tag:               "Core.Description=Product name",
			expectedTerm:      "Org.OData.Core.V1.Description",
			expectedQualifier: "",
			expectedValue:     "Product name",
		},
		{
			name:              "term with hash qualifier",
			tag:               "Core.Description#Short=Product name",
			expectedTerm:      "Org.OData.Core.V1.Description",
			expectedQualifier: "Short",
			expectedValue:     "Product name",
		},
		{
			name:              "term with qualifier segment",
			tag:               "Core.Description=Product name;qualifier=Short",
			expectedTerm:      "Org.OData.Core.V1.Description",
			expectedQualifier: "Short",
			expectedValue:     "Product name",
		},
		{
			name:              "qualifier without value",
			tag:               "Core.Description;qualifier=Short",
			expectedTerm:      "Org.OData.Core.V1.Description",
			expectedQualifier: "Short",
			expectedValue:     true,
		},
		{
			name:        "conflicting qualifiers",
			tag:         "Core.Description#Short;qualifier=Long",
			expectError: true,
		},
		{
			name:        "empty tag",
			tag:         "",
			expectError: true,
		},
		{
			name:        "whitespace only tag",
			tag:         "   ",
			expectError: true,
		},
		{
			name:        "empty term with value",
			tag:         "  =value",
			expectError: true,
		},
		{
			name:        "empty qualifier value",
			tag:         "Core.Description;qualifier=",
			expectError: true,
		},
		{
			name:        "empty term with hash qualifier",
			tag:         "#Qualifier",
			expectError: true,
		},
		{
			name:              "duplicate qualifier specification (matching)",
			tag:               "Core.Description#Short;qualifier=Short",
			expectedTerm:      "Org.OData.Core.V1.Description",
			expectedQualifier: "Short",
			expectedValue:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := ParseAnnotationTag(tt.tag)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if annotation.Term != tt.expectedTerm {
				t.Errorf("Term = %v, want %v", annotation.Term, tt.expectedTerm)
			}

			if annotation.Qualifier != tt.expectedQualifier {
				t.Errorf("Qualifier = %v, want %v", annotation.Qualifier, tt.expectedQualifier)
			}

			if annotation.Value != tt.expectedValue {
				t.Errorf("Value = %v, want %v", annotation.Value, tt.expectedValue)
			}
		})
	}
}

func TestQualifiedTerm(t *testing.T) {
	tests := []struct {
		name       string
		annotation *Annotation
		expected   string
	}{
		{
			name: "term with qualifier",
			annotation: &Annotation{
				Term:      "Org.OData.Core.V1.Description",
				Qualifier: "Short",
			},
			expected: "Org.OData.Core.V1.Description#Short",
		},
		{
			name: "term without qualifier",
			annotation: &Annotation{
				Term: "Org.OData.Core.V1.Computed",
			},
			expected: "Org.OData.Core.V1.Computed",
		},
		{
			name:       "nil annotation",
			annotation: nil,
			expected:   "",
		},
		{
			name: "empty term",
			annotation: &Annotation{
				Term: "",
			},
			expected: "",
		},
		{
			name: "empty term with qualifier",
			annotation: &Annotation{
				Term:      "",
				Qualifier: "Short",
			},
			expected: "",
		},
		{
			name: "term with empty qualifier",
			annotation: &Annotation{
				Term:      "Org.OData.Core.V1.Description",
				Qualifier: "",
			},
			expected: "Org.OData.Core.V1.Description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.annotation.QualifiedTerm()
			if got != tt.expected {
				t.Errorf("QualifiedTerm() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExpandAnnotationAlias(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Core.Computed", "Org.OData.Core.V1.Computed"},
		{"Core.Description", "Org.OData.Core.V1.Description"},
		{"Capabilities.InsertRestrictions", "Org.OData.Capabilities.V1.InsertRestrictions"},
		{"Validation.Pattern", "Org.OData.Validation.V1.Pattern"},
		{"Org.OData.Core.V1.Computed", "Org.OData.Core.V1.Computed"}, // No expansion needed
		{"Custom.MyTerm", "Custom.MyTerm"},                           // Unknown namespace, no expansion
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandAnnotationAlias(tt.input)
			if got != tt.expected {
				t.Errorf("expandAnnotationAlias(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVocabularyAliasMap(t *testing.T) {
	aliases := VocabularyAliasMap()

	if aliases["Org.OData.Core.V1"] != "Core" {
		t.Error("Expected Core alias for Org.OData.Core.V1")
	}
	if aliases["Org.OData.Capabilities.V1"] != "Capabilities" {
		t.Error("Expected Capabilities alias for Org.OData.Capabilities.V1")
	}
	if aliases["Org.OData.Validation.V1"] != "Validation" {
		t.Error("Expected Validation alias for Org.OData.Validation.V1")
	}
}

func TestVocabularyConstants(t *testing.T) {
	// Test that vocabulary constants are correctly defined
	if CoreVocabulary.Namespace != "Org.OData.Core.V1" {
		t.Errorf("CoreVocabulary.Namespace = %s, want Org.OData.Core.V1", CoreVocabulary.Namespace)
	}
	if CoreVocabulary.Alias != "Core" {
		t.Errorf("CoreVocabulary.Alias = %s, want Core", CoreVocabulary.Alias)
	}

	if CapabilitiesVocabulary.Namespace != "Org.OData.Capabilities.V1" {
		t.Errorf("CapabilitiesVocabulary.Namespace = %s, want Org.OData.Capabilities.V1", CapabilitiesVocabulary.Namespace)
	}

	if ValidationVocabulary.Namespace != "Org.OData.Validation.V1" {
		t.Errorf("ValidationVocabulary.Namespace = %s, want Org.OData.Validation.V1", ValidationVocabulary.Namespace)
	}
}
