package handlers

import "fmt"

// GeospatialNotEnabledError is returned when geospatial operations are attempted
// but geospatial features are not enabled on the service
type GeospatialNotEnabledError struct{}

func (e *GeospatialNotEnabledError) Error() string {
	return "geospatial features are not enabled for this service"
}

// IsGeospatialNotEnabledError checks if an error is a GeospatialNotEnabledError
func IsGeospatialNotEnabledError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*GeospatialNotEnabledError)
	if ok {
		return true
	}
	// Also check for wrapped errors
	if unwrapped := fmt.Errorf("%w", err); unwrapped != nil {
		_, ok = unwrapped.(*GeospatialNotEnabledError)
		return ok
	}
	return false
}
