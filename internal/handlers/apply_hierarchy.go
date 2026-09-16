package handlers

import (
	"fmt"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
	"reflect"
)

func containsHierarchy(seq []query.ApplyTransformation) bool {
	for _, tr := range seq {
		if tr.Hierarchy != nil {
			return true
		}
		if tr.Concat != nil {
			for _, branch := range tr.Concat.Sequences {
				if containsHierarchy(branch) {
					return true
				}
			}
		}
	}
	return false
}

// Keep the hierarchy universe separate from the transformed input set.
func (h *EntityHandler) executeHierarchyPipeline(db *gorm.DB, options *query.QueryOptions, meta *metadata.EntityMetadata) ([]map[string]interface{}, error) {
	base := query.ApplyQueryOptions(db.Session(&gorm.Session{}), &query.QueryOptions{Apply: []query.ApplyTransformation{{Type: query.ApplyTypeIdentity}}}, meta, h.logger)
	var universe []map[string]interface{}
	if err := base.Find(&universe).Error; err != nil {
		return nil, err
	}
	rows, err := applyHierarchySequence(universe, universe, options.Apply)
	if err != nil {
		return nil, err
	}
	if options.Compute != nil {
		rows, err = applyMapCompute(rows, options.Compute)
		if err != nil {
			return nil, err
		}
	}
	rows = applyMapFilter(rows, options.Filter)
	if len(options.OrderBy) > 0 {
		applyMapOrderBy(rows, options.OrderBy)
	}
	return applyMapTopSkip(rows, options.Top, options.Skip), nil
}
func applyHierarchySequence(rows, universe []map[string]interface{}, seq []query.ApplyTransformation) ([]map[string]interface{}, error) {
	for _, tr := range seq {
		var err error
		if tr.Hierarchy != nil {
			rows, err = applyHierarchy(rows, universe, tr)
		} else if tr.Concat != nil {
			combined := make([]map[string]interface{}, 0)
			for _, branch := range tr.Concat.Sequences {
				part, e := applyHierarchySequence(cloneApplyRows(rows), universe, branch)
				if e != nil {
					return nil, e
				}
				combined = append(combined, part...)
			}
			rows = combined
		} else {
			rows, err = applySupportedTailTransformations(rows, []query.ApplyTransformation{tr})
		}
		if err != nil {
			return nil, err
		}
	}
	return rows, nil
}
func cloneApplyRows(rows []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		out[i] = make(map[string]interface{}, len(row))
		for k, v := range row {
			out[i][k] = v
		}
	}
	return out
}
func hierarchyKey(value interface{}) string {
	if value == nil {
		return ""
	}
	v := reflect.ValueOf(value)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	return fmt.Sprint(v.Interface())
}
func applyHierarchy(rows, universe []map[string]interface{}, tr query.ApplyTransformation) ([]map[string]interface{}, error) {
	spec := tr.Hierarchy
	nodes := make(map[string]map[string]interface{}, len(universe))
	parents := make(map[string]string, len(universe))
	children := make(map[string][]string)
	ordered := cloneApplyRows(universe)
	if len(spec.OrderBy) > 0 {
		applyMapOrderBy(ordered, spec.OrderBy)
	}
	for _, row := range ordered {
		key := hierarchyKey(row[spec.NodeProperty])
		if key == "" {
			return nil, fmt.Errorf("null hierarchy node identifier")
		}
		if _, ok := nodes[key]; ok {
			return nil, fmt.Errorf("duplicate hierarchy node identifier")
		}
		nodes[key] = row
		parents[key] = hierarchyKey(row[spec.ParentProperty])
	}
	roots := make([]string, 0)
	for _, row := range ordered {
		key := hierarchyKey(row[spec.NodeProperty])
		parent := parents[key]
		if _, ok := nodes[parent]; !ok {
			roots = append(roots, key)
		} else {
			children[parent] = append(children[parent], key)
		}
	}
	// Iterative three-color validation avoids recursion on deep trees.
	colors := make(map[string]uint8, len(nodes))
	for start := range nodes {
		path := make([]string, 0)
		current := start
		for nodes[current] != nil && colors[current] == 0 {
			colors[current] = 1
			path = append(path, current)
			current = parents[current]
		}
		if colors[current] == 1 {
			return nil, fmt.Errorf("recursive hierarchy contains a cycle")
		}
		for _, key := range path {
			colors[key] = 2
		}
	}
	result := make([]map[string]interface{}, 0)
	if tr.Type == query.ApplyTypeTraverse {
		byNode := make(map[string][]map[string]interface{})
		for _, row := range rows {
			key := hierarchyKey(row[spec.Property])
			byNode[key] = append(byNode[key], row)
		}
		type visit struct {
			key  string
			emit bool
		}
		stack := make([]visit, 0)
		for i := len(roots) - 1; i >= 0; i-- {
			stack = append(stack, visit{key: roots[i]})
		}
		for len(stack) > 0 {
			entry := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if entry.emit || !spec.PostOrder {
				result = append(result, byNode[entry.key]...)
			}
			if entry.emit {
				continue
			}
			if spec.PostOrder {
				stack = append(stack, visit{key: entry.key, emit: true})
			}
			list := children[entry.key]
			for i := len(list) - 1; i >= 0; i-- {
				stack = append(stack, visit{key: list[i]})
			}
		}
		return result, nil
	}
	starts, err := applyHierarchySequence(cloneApplyRows(rows), universe, spec.Start)
	if err != nil {
		return nil, err
	}
	matched := make(map[string]bool)
	for _, row := range starts {
		start := hierarchyKey(row[spec.Property])
		if nodes[start] == nil {
			continue
		}
		if spec.KeepStart {
			matched[start] = true
		}
		frontier := []string{start}
		seen := map[string]bool{start: true}
		for distance := 1; len(frontier) > 0 && (spec.Distance < 0 || distance <= spec.Distance); distance++ {
			next := make([]string, 0)
			for _, node := range frontier {
				neighbors := children[node]
				if tr.Type == query.ApplyTypeAncestors {
					neighbors = []string{parents[node]}
				}
				for _, key := range neighbors {
					if nodes[key] != nil && !seen[key] {
						seen[key] = true
						matched[key] = true
						next = append(next, key)
					}
				}
			}
			frontier = next
		}
	}
	for _, row := range rows {
		if matched[hierarchyKey(row[spec.Property])] {
			result = append(result, row)
		}
	}
	return result, nil
}
