package query

import "testing"

// TestParseApply_TransformationCatalog_AllSupported defines the expected
// transformation catalog for $apply according to issue #630 scope.
func TestParseApply_TransformationCatalog_AllSupported(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name         string
		apply        string
		expectedType ApplyTransformationType
	}{
		{name: "identity", apply: "identity", expectedType: ApplyTypeIdentity},
		{name: "groupby", apply: "groupby((Category))", expectedType: ApplyTypeGroupBy},
		{name: "aggregate", apply: "aggregate(Price with sum as Total)", expectedType: ApplyTypeAggregate},
		{name: "filter", apply: "filter(Price gt 10)", expectedType: ApplyTypeFilter},
		{name: "compute", apply: "compute(Price mul Quantity as Revenue)", expectedType: ApplyTypeCompute},
		{name: "orderby", apply: "orderby(Price desc)", expectedType: ApplyTypeOrderBy},
		{name: "skip", apply: "skip(1)", expectedType: ApplyTypeSkip},
		{name: "top", apply: "top(2)", expectedType: ApplyTypeTop},
		{name: "search", apply: "search(Laptop)", expectedType: ApplyTransformationType("search")},
		{name: "concat", apply: "concat(filter(Price gt 100),filter(Price le 100))", expectedType: ApplyTransformationType("concat")},
		{name: "join", apply: "join(Lines as Line)", expectedType: ApplyTransformationType("join")},
		{name: "outerjoin", apply: "outerjoin(Lines as Line)", expectedType: ApplyTransformationType("outerjoin")},
		{name: "topcount", apply: "topcount(2,Price)", expectedType: ApplyTransformationType("topcount")},
		{name: "bottomcount", apply: "bottomcount(2,Price)", expectedType: ApplyTransformationType("bottomcount")},
		{name: "toppercent", apply: "toppercent(100,Price)", expectedType: ApplyTransformationType("toppercent")},
		{name: "bottompercent", apply: "bottompercent(100,Price)", expectedType: ApplyTransformationType("bottompercent")},
		{name: "topsum", apply: "topsum(100000,Price)", expectedType: ApplyTransformationType("topsum")},
		{name: "bottomsum", apply: "bottomsum(100000,Price)", expectedType: ApplyTransformationType("bottomsum")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans, err := parseApply(tt.apply, meta, 0)
			if err != nil {
				t.Fatalf("parseApply(%q) failed: %v", tt.apply, err)
			}
			if len(trans) != 1 {
				t.Fatalf("parseApply(%q) expected one transformation, got %d", tt.apply, len(trans))
			}
			if trans[0].Type != tt.expectedType {
				t.Fatalf("parseApply(%q) expected type %q, got %q", tt.apply, tt.expectedType, trans[0].Type)
			}
		})
	}
}

// TestParseApply_TransformationCatalog_Unsupported verifies that transformations
// not supported by the library (hierarchy traversal, service-defined functions)
// return parse errors rather than silently succeeding.
func TestParseApply_TransformationCatalog_Unsupported(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name  string
		apply string
	}{
		{name: "ancestors-empty", apply: "ancestors()"},
		{name: "descendants-empty", apply: "descendants()"},
		{name: "traverse-empty", apply: "traverse()"},
		{name: "service-defined-function", apply: "Default.CustomSetTransform()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseApply(tt.apply, meta, 0)
			if err == nil {
				t.Fatalf("parseApply(%q) expected an error but got none", tt.apply)
			}
		})
	}
}

// TestParseApply_GroupBySecondParameter_TransformationSequence ensures the
// second parameter of groupby supports a slash-separated transformation sequence.
func TestParseApply_GroupBySecondParameter_TransformationSequence(t *testing.T) {
	meta := getApplyTestMetadata(t)

	apply := "groupby((Category),aggregate($count as GroupCount)/filter(GroupCount gt 0)/orderby(GroupCount desc)/skip(0)/top(2))"
	trans, err := parseApply(apply, meta, 0)
	if err != nil {
		t.Fatalf("parseApply(%q) failed: %v", apply, err)
	}

	if len(trans) != 1 {
		t.Fatalf("expected 1 top-level transformation, got %d", len(trans))
	}
	if trans[0].Type != ApplyTypeGroupBy || trans[0].GroupBy == nil {
		t.Fatalf("expected top-level groupby transformation, got %+v", trans[0])
	}

	nested := trans[0].GroupBy.Transform
	if len(nested) != 5 {
		t.Fatalf("expected 5 nested transformations, got %d", len(nested))
	}

	expected := []ApplyTransformationType{
		ApplyTypeAggregate,
		ApplyTypeFilter,
		ApplyTypeOrderBy,
		ApplyTypeSkip,
		ApplyTypeTop,
	}
	for i := range expected {
		if nested[i].Type != expected[i] {
			t.Fatalf("nested[%d] expected %q, got %q", i, expected[i], nested[i].Type)
		}
	}
}
