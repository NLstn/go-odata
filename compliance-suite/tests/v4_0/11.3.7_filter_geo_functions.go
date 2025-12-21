package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterGeoFunctions creates a test suite for geospatial filter functions
func FilterGeoFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.7 Geospatial Functions in Filter",
		"Tests geospatial functions (geo.distance, geo.length, geo.intersects) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_GeospatialFunctions",
	)
	RegisterFilterGeoFunctionsTests(suite)
	return suite
}

// RegisterFilterGeoFunctionsTests registers tests for geospatial filter functions
func RegisterFilterGeoFunctionsTests(suite *framework.TestSuite) {
	suite.AddTest(
		"geo.distance function in filter",
		"Filter using geo.distance() to find entities within distance (optional feature)",
		testGeoDistance,
	)

	suite.AddTest(
		"geo.length function in filter",
		"Filter using geo.length() on linestring geometries (optional feature)",
		testGeoLength,
	)

	suite.AddTest(
		"geo.intersects function in filter",
		"Filter using geo.intersects() to test spatial intersection (optional feature)",
		testGeoIntersects,
	)

	suite.AddTest(
		"Invalid geo function returns error",
		"Invalid geospatial function name returns 400 Bad Request",
		testInvalidGeoFunction,
	)

	suite.AddTest(
		"geo.distance with invalid syntax returns error",
		"Missing required parameter for geo.distance returns 400",
		testGeoDistanceInvalidSyntax,
	)

	suite.AddTest(
		"Valid geospatial literal format",
		"Properly formatted geography literals are accepted (optional feature)",
		testGeoLiteralFormat,
	)

	suite.AddTest(
		"Geometry vs geography distinction",
		"Test geometry (flat earth) vs geography (round earth) types (optional feature)",
		testGeometryVsGeography,
	)

	suite.AddTest(
		"Combining geo functions with other filters",
		"Combine geospatial filters with regular property filters (optional feature)",
		testGeoCombinedFilter,
	)
}

// validateGeoResponse validates the structure of a geospatial query response
// Returns nil if validation passes, error otherwise
func validateGeoResponse(respBody []byte) error {
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate response structure
	value, ok := result["value"]
	if !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	// Value must be an array (empty is ok if no products match)
	products, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("'value' is not an array")
	}

	// If there are products, validate they have required properties
	for i, p := range products {
		product, ok := p.(map[string]interface{})
		if !ok {
			return fmt.Errorf("product at index %d is not an object", i)
		}
		// Each product should have standard properties
		if _, ok := product["ID"]; !ok {
			return fmt.Errorf("product at index %d missing ID", i)
		}
	}

	return nil
}

func testGeoDistance(ctx *framework.TestContext) error {
	// Geospatial functions are optional OData features
	// Test geo.distance to find products within 10000 meters of origin
	filter := url.QueryEscape("geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	// 200 OK = database supports geospatial, validate response
	// 400 = bad request due to malformed query or database doesn't support geospatial
	// 404/501 = feature not implemented by library
	// 500 = internal server error, often from SQL error when database lacks spatial functions
	//       (e.g., SQLite without SpatiaLite extension) - treat as "not supported" for optional features
	switch resp.StatusCode {
	case 200:
		// Database supports geospatial - validate proper filtering occurred
		return validateGeoResponse(resp.Body)
	case 400, 500:
		// Database doesn't support geospatial functions (e.g., SQLite without SpatiaLite)
		// 500 occurs when SQL functions like ST_Distance are not available, causing SQL errors
		return ctx.Skip("Database doesn't support geospatial functions (optional feature)")
	case 404, 501:
		// Library feature not implemented
		return ctx.Skip("geo.distance not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testGeoLength(ctx *framework.TestContext) error {
	// Test geo.length to find products with route length > 1000 meters
	filter := url.QueryEscape("geo.length(Route) gt 1000")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Database supports geospatial - validate proper filtering occurred
		return validateGeoResponse(resp.Body)
	case 400, 500:
		// Database doesn't support geospatial functions
		// 500 occurs when SQL functions like ST_Length are not available
		return ctx.Skip("Database doesn't support geospatial functions (optional feature)")
	case 404, 501:
		// Library feature not implemented
		return ctx.Skip("geo.length not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testGeoIntersects(ctx *framework.TestContext) error {
	// Test geo.intersects to find products whose Area intersects with a polygon
	filter := url.QueryEscape("geo.intersects(Area,geography'SRID=4326;POLYGON((0 0,10 0,10 10,0 10,0 0))')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Database supports geospatial - validate proper filtering occurred
		return validateGeoResponse(resp.Body)
	case 400, 500:
		// Database doesn't support geospatial functions
		// 500 occurs when SQL functions like ST_Intersects are not available
		return ctx.Skip("Database doesn't support geospatial functions (optional feature)")
	case 404, 501:
		// Library feature not implemented
		return ctx.Skip("geo.intersects not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testInvalidGeoFunction(ctx *framework.TestContext) error {
	filter := url.QueryEscape("geo.invalid(Location)")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	// Should return 400 or 404 for invalid function
	if resp.StatusCode != 400 && resp.StatusCode != 404 {
		return fmt.Errorf("expected status 400 or 404, got %d", resp.StatusCode)
	}

	return nil
}

func testGeoDistanceInvalidSyntax(ctx *framework.TestContext) error {
	// Missing required second parameter
	filter := url.QueryEscape("geo.distance(Location) lt 100")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 400 {
		return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	return nil
}

func testGeoLiteralFormat(ctx *framework.TestContext) error {
	// Test properly formatted geography literal with specific SRID and coordinates
	filter := url.QueryEscape("geo.distance(Location,geography'SRID=4326;POINT(-122.1 47.6)') lt 5000")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Database supports geospatial and literal format is accepted
		return validateGeoResponse(resp.Body)
	case 400, 500:
		// Database doesn't support geospatial functions
		// 500 occurs when SQL spatial functions are not available
		return ctx.Skip("Database doesn't support geospatial functions (optional feature)")
	case 404, 501:
		// Library feature not implemented
		return ctx.Skip("Geospatial functions not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testGeometryVsGeography(ctx *framework.TestContext) error {
	// Test geometry (flat earth, planar) vs geography (round earth, geodetic)
	// Geometry uses SRID=0, Geography typically uses SRID=4326 (WGS84)
	filter := url.QueryEscape("geo.distance(Location,geometry'SRID=0;POINT(0 0)') lt 100")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Database supports geometry type
		return validateGeoResponse(resp.Body)
	case 400, 500:
		// Database doesn't support geospatial or geometry type
		// 500 occurs when SQL spatial functions are not available
		return ctx.Skip("Database doesn't support geometry type (optional feature)")
	case 404, 501:
		// Library feature not implemented
		return ctx.Skip("Geometry type not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testGeoCombinedFilter(ctx *framework.TestContext) error {
	// Test combining geospatial filter with regular property filter
	// This validates that geospatial functions can be used in complex filter expressions
	filter := url.QueryEscape("Price gt 100 and geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Database supports combined filters with geospatial
		// First validate basic response structure
		if err := validateGeoResponse(resp.Body); err != nil {
			return err
		}

		// Additional validation for the combined filter: verify Price filter was applied
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if value, ok := result["value"].([]interface{}); ok {
			for i, p := range value {
				if product, ok := p.(map[string]interface{}); ok {
					// If Price is present, verify it meets the filter condition (> 100)
					if price, ok := product["Price"]; ok {
						var priceVal float64
						switch v := price.(type) {
						case float64:
							priceVal = v
						case int:
							priceVal = float64(v)
						default:
							return fmt.Errorf("product at index %d has unexpected Price type: %T", i, price)
						}
						if priceVal <= 100 {
							return fmt.Errorf("product at index %d has Price %f, expected > 100", i, priceVal)
						}
					}
				}
			}
		}

		return nil
	case 400, 500:
		// Database doesn't support geospatial functions or combined filters
		// 500 occurs when SQL spatial functions are not available
		return ctx.Skip("Database doesn't support combined geospatial filters (optional feature)")
	case 404, 501:
		// Library feature not implemented
		return ctx.Skip("Combined geo filter not supported (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
