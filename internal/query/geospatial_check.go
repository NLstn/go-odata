package query

// ContainsGeospatialOperations checks if a filter expression contains any geospatial operations
func ContainsGeospatialOperations(filter *FilterExpression) bool {
	if filter == nil {
		return false
	}

	// Check current operator
	switch filter.Operator {
	case OpGeoDistance, OpGeoLength, OpGeoIntersects:
		return true
	}

	// Recursively check logical expressions
	if filter.Left != nil && ContainsGeospatialOperations(filter.Left) {
		return true
	}
	if filter.Right != nil && ContainsGeospatialOperations(filter.Right) {
		return true
	}

	return false
}
