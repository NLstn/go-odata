package query

import (
	"fmt"
	"github.com/nlstn/go-odata/internal/metadata"
	"strconv"
	"strings"
)

func annotationPath(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]interface{}); ok {
		for _, k := range []string{"$PropertyPath", "$NavigationPropertyPath"} {
			if s, ok := m[k].(string); ok {
				return s
			}
		}
	}
	return ""
}
func parseHierarchyTransformationWithMetadata(text string, kind ApplyTransformationType, meta *metadata.EntityMetadata, maxIn int, insensitive bool) (*ApplyTransformation, error) {
	prefix := string(kind) + "("
	if !strings.HasSuffix(text, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in %s", kind)
	}
	args := splitAggregateExpressions(text[len(prefix) : len(text)-1])
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
	}
	if len(args) < 4 || meta == nil {
		return nil, fmt.Errorf("%s requires hierarchy, qualifier, node path and selection", kind)
	}
	if args[0] != "$root/"+meta.EntitySetName {
		return nil, fmt.Errorf("%s: only a hierarchy on the current entity set is supported", kind)
	}
	var record map[string]interface{}
	for _, a := range meta.Annotations.GetByTerm("Org.OData.Aggregation.V1.RecursiveHierarchy") {
		if a.Qualifier == args[1] {
			var ok bool
			record, ok = a.Value.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid hierarchy annotation")
			}
			break
		}
	}
	if record == nil {
		return nil, fmt.Errorf("unknown hierarchy qualifier %q", args[1])
	}
	node := meta.FindStructuralProperty(annotationPath(record["NodeProperty"]))
	parent := meta.FindNavigationProperty(annotationPath(record["ParentNavigationProperty"]))
	property := meta.FindStructuralProperty(args[2])
	if node == nil || parent == nil || property == nil || node.IsComplexType || node.IsStream || property.IsComplexType || property.IsStream || parent.NavigationIsArray || parent.NavigationTarget != meta.EntityName {
		return nil, fmt.Errorf("%s requires direct primitive node paths and a single parent navigation on the same type", kind)
	}
	parentProperty := ""
	for dependent, principal := range parent.ReferentialConstraints {
		if principal == node.Name || principal == node.JsonName {
			parentProperty = dependent
		}
	}
	if parentProperty == "" && node.IsKey && len(meta.KeyProperties) == 1 {
		for _, p := range meta.Properties {
			if p.ColumnName == parent.ForeignKeyColumnName {
				parentProperty = p.JsonName
			}
		}
	}
	if parentProperty == "" {
		return nil, fmt.Errorf("cannot resolve parent reference")
	}
	h := &HierarchyTransformation{Nodes: args[0], Qualifier: args[1], Property: args[2], NodeProperty: node.JsonName, ParentProperty: parentProperty, Distance: -1}
	if kind == ApplyTypeTraverse {
		if args[3] != "preorder" && args[3] != "postorder" {
			return nil, fmt.Errorf("traverse order must be preorder or postorder")
		}
		h.PostOrder = args[3] == "postorder"
		if args[2] != node.JsonName {
			return nil, fmt.Errorf("traverse requires the hierarchy node property as input path")
		}
		if len(args) > 4 {
			var err error
			h.OrderBy, err = parseOrderBy(strings.Join(args[4:], ","), meta, nil)
			if err != nil {
				return nil, err
			}
		}
	} else {
		if len(args) > 6 {
			return nil, fmt.Errorf("too many hierarchy parameters")
		}
		var err error
		h.Start, err = parseApplyWithCaseSensitivity(args[3], meta, maxIn, insensitive)
		if err != nil {
			return nil, err
		}
		for _, tr := range h.Start {
			switch tr.Type {
			case ApplyTypeFilter, ApplyTypeOrderBy, ApplyTypeSkip, ApplyTypeTop, ApplyTypeSearch, ApplyTypeTopCount, ApplyTypeBottomCount, ApplyTypeTopPercent, ApplyTypeBottomPercent, ApplyTypeTopSum, ApplyTypeBottomSum, ApplyTypeAncestors, ApplyTypeDescendants:
			default:
				return nil, fmt.Errorf("hierarchy start sequence must produce a subset")
			}
		}
		rest := args[4:]
		if len(rest) > 0 && rest[len(rest)-1] == "keep start" {
			h.KeepStart = true
			rest = rest[:len(rest)-1]
		}
		if len(rest) > 1 {
			return nil, fmt.Errorf("invalid hierarchy parameters")
		}
		if len(rest) == 1 {
			h.Distance, err = strconv.Atoi(rest[0])
			if err != nil || h.Distance < 1 {
				return nil, fmt.Errorf("hierarchy distance must be positive")
			}
		}
	}
	return &ApplyTransformation{Type: kind, Hierarchy: h}, nil
}
