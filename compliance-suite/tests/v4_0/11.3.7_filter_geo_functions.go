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

func testGeoDistance(ctx *framework.TestContext) error {
	// Geospatial functions are optional OData features
	filter := url.QueryEscape("geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	// 200 OK = supported, 400/404/501 = not implemented (acceptable)
	switch resp.StatusCode {
	case 200:
		// Validate response structure when geospatial is supported
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if _, ok := result["value"]; !ok {
			return fmt.Errorf("response missing 'value' array")
		}
		return nil
	case 400, 404, 501:
		return ctx.Skip("geo.distance not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testGeoLength(ctx *framework.TestContext) error {
	filter := url.QueryEscape("geo.length(Route) gt 1000")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Validate response structure when geospatial is supported
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if _, ok := result["value"]; !ok {
			return fmt.Errorf("response missing 'value' array")
		}
		return nil
	case 400, 404, 501:
		return ctx.Skip("geo.length not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testGeoIntersects(ctx *framework.TestContext) error {
	filter := url.QueryEscape("geo.intersects(Area,geography'SRID=4326;POLYGON((0 0,10 0,10 10,0 10,0 0))')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Validate response structure when geospatial is supported
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if _, ok := result["value"]; !ok {
			return fmt.Errorf("response missing 'value' array")
		}
		return nil
	case 400, 404, 501:
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
	filter := url.QueryEscape("geo.distance(Location,geography'SRID=4326;POINT(-122.1 47.6)') lt 5000")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Validate response structure when geospatial is supported
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if _, ok := result["value"]; !ok {
			return fmt.Errorf("response missing 'value' array")
		}
		return nil
	case 400, 404, 501:
		return ctx.Skip("Geospatial functions not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testGeometryVsGeography(ctx *framework.TestContext) error {
	// Test geometry (flat earth) type
	filter := url.QueryEscape("geo.distance(Location,geometry'SRID=0;POINT(0 0)') lt 100")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Validate response structure when geospatial is supported
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if _, ok := result["value"]; !ok {
			return fmt.Errorf("response missing 'value' array")
		}
		return nil
	case 400, 404, 501:
		return ctx.Skip("Geometry type not implemented (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func testGeoCombinedFilter(ctx *framework.TestContext) error {
	filter := url.QueryEscape("Price gt 100 and geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		// Validate response structure when geospatial is supported
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if _, ok := result["value"]; !ok {
			return fmt.Errorf("response missing 'value' array")
		}
		return nil
	case 400, 404, 501:
		return ctx.Skip("Combined geo filter not supported (optional feature)")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
