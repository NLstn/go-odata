package query

import (
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// applySelectToExpandedEntity applies select to an expanded navigation property (single entity or collection)
// This is a simplified version that doesn't require full metadata - it works with reflection
func applySelectToExpandedEntity(expandedValue interface{}, selectedProps []string, expandOptions []ExpandOption) interface{} {
	if len(selectedProps) == 0 || expandedValue == nil {
		return expandedValue
	}

	selectedPropMap := make(map[string]bool)
	for _, propName := range selectedProps {
		selectedPropMap[strings.TrimSpace(propName)] = true
	}

	val := reflect.ValueOf(expandedValue)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return expandedValue
		}
		val = val.Elem()
	}

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		resultSlice := make([]map[string]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			itemVal := val.Index(i)
			resultSlice[i] = filterEntityFields(itemVal, selectedPropMap, expandOptions)
		}
		return resultSlice
	}

	if val.Kind() == reflect.Struct {
		return filterEntityFields(val, selectedPropMap, expandOptions)
	}

	return expandedValue
}

// filterEntityFields filters struct fields based on selected properties map
func filterEntityFields(entityVal reflect.Value, selectedPropMap map[string]bool, expandOptions []ExpandOption) map[string]interface{} {
	filtered := make(map[string]interface{})
	entityType := entityVal.Type()

	idFields := []string{"ID", "Id", "id"}

	// Build map of expanded navigation properties by name
	expandedNavMap := make(map[string]*ExpandOption)
	for i := range expandOptions {
		expandedNavMap[expandOptions[i].NavigationProperty] = &expandOptions[i]
	}

	for i := 0; i < entityVal.NumField(); i++ {
		field := entityType.Field(i)
		fieldVal := entityVal.Field(i)

		if !fieldVal.CanInterface() {
			continue
		}

		jsonName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				jsonName = parts[0]
			}
		}

		isSelected := selectedPropMap[field.Name] || selectedPropMap[jsonName]
		isKeyField := false
		for _, keyName := range idFields {
			if field.Name == keyName {
				isKeyField = true
				break
			}
		}

		if odataTag := field.Tag.Get("odata"); odataTag != "" {
			if strings.Contains(odataTag, "key") {
				isKeyField = true
			}
		}

		// Check if this field is a navigation property that should be expanded
		var matchedExpandOpt *ExpandOption
		if opt, ok := expandedNavMap[field.Name]; ok {
			matchedExpandOpt = opt
		} else if opt, ok := expandedNavMap[jsonName]; ok {
			matchedExpandOpt = opt
		}

		if isSelected || isKeyField {
			filtered[jsonName] = fieldVal.Interface()
		} else if matchedExpandOpt != nil {
			// Include expanded navigation properties and recursively apply their select/expand
			val := fieldVal.Interface()
			if matchedExpandOpt.Select != nil && len(matchedExpandOpt.Select) > 0 && val != nil {
				val = applySelectToExpandedEntity(val, matchedExpandOpt.Select, matchedExpandOpt.Expand)
			}
			filtered[jsonName] = val
		}
	}

	return filtered
}

// evaluateSingleComputeExpression evaluates a single compute expression against an entity value
func evaluateSingleComputeExpression(entityVal reflect.Value, expr *FilterExpression) interface{} {
	if expr == nil {
		return nil
	}

	// Handle arithmetic expressions (mul, div, add, sub, mod)
	if expr.Left != nil && expr.Right != nil && expr.Logical != "" {
		leftVal := evaluateSingleComputeExpression(entityVal, expr.Left)
		rightVal := evaluateSingleComputeExpression(entityVal, expr.Right)

		leftFloat := toFloat64(leftVal)
		rightFloat := toFloat64(rightVal)

		switch expr.Logical {
		case "mul":
			return leftFloat * rightFloat
		case "div":
			if rightFloat != 0 {
				return leftFloat / rightFloat
			}
			return nil
		case "add":
			return leftFloat + rightFloat
		case "sub":
			return leftFloat - rightFloat
		case "mod":
			if rightFloat != 0 {
				return int64(leftFloat) % int64(rightFloat)
			}
			return nil
		}
	}

	// Handle arithmetic operators (OpMul, OpDiv, etc.)
	if expr.Left != nil && expr.Right != nil && expr.Operator != "" {
		leftVal := evaluateSingleComputeExpression(entityVal, expr.Left)
		rightVal := evaluateSingleComputeExpression(entityVal, expr.Right)

		leftFloat := toFloat64(leftVal)
		rightFloat := toFloat64(rightVal)

		switch expr.Operator {
		case OpMul:
			return leftFloat * rightFloat
		case OpDiv:
			if rightFloat != 0 {
				return leftFloat / rightFloat
			}
			return nil
		case OpAdd:
			return leftFloat + rightFloat
		case OpSub:
			return leftFloat - rightFloat
		case OpMod:
			if rightFloat != 0 {
				return int64(leftFloat) % int64(rightFloat)
			}
			return nil
		}
	}

	// Handle property reference
	if expr.Property != "" && expr.Left == nil && expr.Right == nil {
		return getPropertyValue(entityVal, expr.Property)
	}

	// Handle literal value
	if expr.Value != nil && expr.Property == "" && expr.Left == nil && expr.Right == nil {
		return expr.Value
	}

	return nil
}

// getPropertyValue gets a property value from an entity using reflection
func getPropertyValue(entityVal reflect.Value, propName string) interface{} {
	if !entityVal.IsValid() {
		return nil
	}

	if entityVal.Kind() == reflect.Ptr {
		if entityVal.IsNil() {
			return nil
		}
		entityVal = entityVal.Elem()
	}

	if entityVal.Kind() != reflect.Struct {
		return nil
	}

	entityType := entityVal.Type()

	// Try direct field name
	for i := 0; i < entityVal.NumField(); i++ {
		field := entityType.Field(i)
		fieldVal := entityVal.Field(i)

		jsonName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				jsonName = parts[0]
			}
		}

		if field.Name == propName || jsonName == propName {
			if fieldVal.CanInterface() {
				return fieldVal.Interface()
			}
		}
	}

	return nil
}

// toFloat64 converts an interface value to float64
func toFloat64(val interface{}) float64 {
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	default:
		return 0
	}
}

// ApplyExpandComputeToResults processes results to add computed properties to expanded entities
// This should be called after fetching results but before response formatting
func ApplyExpandComputeToResults(results interface{}, expandOptions []ExpandOption) interface{} {
	if len(expandOptions) == 0 || results == nil {
		return results
	}

	// Check if any expand option has compute
	hasCompute := false
	for _, opt := range expandOptions {
		if opt.Compute != nil && len(opt.Compute.Expressions) > 0 {
			hasCompute = true
			break
		}
	}

	if !hasCompute {
		return results
	}

	// Process the results
	val := reflect.ValueOf(results)

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		// Convert to slice of maps with computed properties
		resultMaps := make([]map[string]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)
			resultMaps[i] = convertEntityToMapWithCompute(item, expandOptions)
		}
		return resultMaps
	}

	return results
}

// convertEntityToMapWithCompute converts an entity to a map, adding computed properties to expanded entities
func convertEntityToMapWithCompute(entityVal reflect.Value, expandOptions []ExpandOption) map[string]interface{} {
	result := make(map[string]interface{})

	if !entityVal.IsValid() {
		return result
	}

	if entityVal.Kind() == reflect.Ptr {
		if entityVal.IsNil() {
			return result
		}
		entityVal = entityVal.Elem()
	}

	if entityVal.Kind() != reflect.Struct {
		return result
	}

	entityType := entityVal.Type()

	// Build a map of expand options by navigation property name
	expandOptByName := make(map[string]*ExpandOption)
	for i := range expandOptions {
		expandOptByName[expandOptions[i].NavigationProperty] = &expandOptions[i]
	}

	for i := 0; i < entityVal.NumField(); i++ {
		field := entityType.Field(i)
		fieldVal := entityVal.Field(i)

		if !fieldVal.CanInterface() {
			continue
		}

		jsonName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				jsonName = parts[0]
			}
		}

		// Check if this is an expanded navigation property with compute
		if expandOpt, ok := expandOptByName[jsonName]; ok {
			if expandOpt.Compute != nil && len(expandOpt.Compute.Expressions) > 0 {
				// Apply compute to this navigation property
				result[jsonName] = applyComputeToNavPropertyValue(fieldVal, expandOpt.Compute)
				continue
			}
		}
		if expandOpt, ok := expandOptByName[field.Name]; ok {
			if expandOpt.Compute != nil && len(expandOpt.Compute.Expressions) > 0 {
				result[jsonName] = applyComputeToNavPropertyValue(fieldVal, expandOpt.Compute)
				continue
			}
		}

		result[jsonName] = fieldVal.Interface()
	}

	return result
}

// applyComputeToNavPropertyValue applies compute to a navigation property value
func applyComputeToNavPropertyValue(fieldVal reflect.Value, compute *ComputeTransformation) interface{} {
	if !fieldVal.IsValid() || compute == nil {
		if fieldVal.IsValid() && fieldVal.CanInterface() {
			return fieldVal.Interface()
		}
		return nil
	}

	if fieldVal.Kind() == reflect.Ptr {
		if fieldVal.IsNil() {
			return nil
		}
		fieldVal = fieldVal.Elem()
	}

	// Single entity
	if fieldVal.Kind() == reflect.Struct {
		return convertNavEntityToMapWithCompute(fieldVal, compute)
	}

	// Collection
	if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
		results := make([]map[string]interface{}, fieldVal.Len())
		for i := 0; i < fieldVal.Len(); i++ {
			item := fieldVal.Index(i)
			if item.Kind() == reflect.Ptr && !item.IsNil() {
				item = item.Elem()
			}
			if item.Kind() == reflect.Struct {
				results[i] = convertNavEntityToMapWithCompute(item, compute)
			}
		}
		return results
	}

	if fieldVal.CanInterface() {
		return fieldVal.Interface()
	}
	return nil
}

// convertNavEntityToMapWithCompute converts a navigation entity to a map with computed properties
func convertNavEntityToMapWithCompute(entityVal reflect.Value, compute *ComputeTransformation) map[string]interface{} {
	result := make(map[string]interface{})

	if !entityVal.IsValid() {
		return result
	}

	if entityVal.Kind() == reflect.Ptr {
		if entityVal.IsNil() {
			return result
		}
		entityVal = entityVal.Elem()
	}

	if entityVal.Kind() != reflect.Struct {
		return result
	}

	entityType := entityVal.Type()

	// Copy all existing fields
	for i := 0; i < entityVal.NumField(); i++ {
		field := entityType.Field(i)
		fieldVal := entityVal.Field(i)

		if !fieldVal.CanInterface() {
			continue
		}

		jsonName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				jsonName = parts[0]
			}
		}

		result[jsonName] = fieldVal.Interface()
	}

	// Add computed properties
	for _, expr := range compute.Expressions {
		value := evaluateSingleComputeExpression(entityVal, expr.Expression)
		result[expr.Alias] = value
	}

	return result
}

// findProperty finds a property by name or JSON name in the entity metadata
func findProperty(propName string, entityMetadata *metadata.EntityMetadata) *metadata.PropertyMetadata {
	if entityMetadata == nil {
		return nil
	}
	return entityMetadata.FindProperty(propName)
}

// sanitizeIdentifier sanitizes a user-provided identifier to prevent SQL injection.
// It ensures that the identifier contains only safe characters (alphanumeric and underscore).
// Returns an empty string if the identifier contains invalid characters.
// Special OData identifiers like "$it" are allowed.
func sanitizeIdentifier(identifier string) string {
	if identifier == "" {
		return ""
	}

	if identifier == "$it" || identifier == "$count" {
		return identifier
	}

	if len(identifier) > 1 && identifier[0] == '$' {
		for i, ch := range identifier[1:] {
			if i == 0 {
				if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && ch != '_' {
					return ""
				}
			} else {
				if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
					return ""
				}
			}
		}
		return identifier
	}

	for i, ch := range identifier {
		if i == 0 {
			if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && ch != '_' {
				return ""
			}
		} else {
			if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
				return ""
			}
		}
	}

	upper := strings.ToUpper(identifier)
	reservedKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE",
		"UNION", "JOIN", "WHERE", "FROM", "INTO", "VALUES", "SET", "EXEC", "EXECUTE",
	}
	for _, keyword := range reservedKeywords {
		if upper == keyword {
			return ""
		}
	}

	return identifier
}
