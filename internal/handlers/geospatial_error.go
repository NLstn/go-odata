package handlers

import "errors"

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
	var geoErr *GeospatialNotEnabledError
	return errors.As(err, &geoErr)
}
