package v4_01

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QuerySchemaVersion creates the 11.2.12 $schemaversion query option test suite.
func QuerySchemaVersion() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.12 System Query Option $schemaversion",
		"Validates OData 4.01 $schemaversion behavior when schema versioning is advertised.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionschemaversion",
	)

	var cachedVersion string

	getSchemaVersion := func(ctx *framework.TestContext) (string, error) {
		if cachedVersion != "" {
			return cachedVersion, nil
		}

		resp, err := ctx.GET("/$metadata")
		if err != nil {
			return "", err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return "", err
		}

		metadata := string(resp.Body)
		if !strings.Contains(metadata, "Core.SchemaVersion") {
			return "", ctx.Skip("service metadata does not advertise Core.SchemaVersion; $schemaversion version-binding semantics are not applicable")
		}

		re := regexp.MustCompile(`Core\.SchemaVersion\"\s+String=\"([^\"]+)\"|Core\.SchemaVersion\"\s+AnnotationPath=\"([^\"]+)\"|Term=\"Core\.SchemaVersion\"\s+String=\"([^\"]+)\"`)
		match := re.FindStringSubmatch(metadata)
		if len(match) == 0 {
			return "", ctx.Skip("Core.SchemaVersion is present but no concrete String value was found in metadata")
		}

		for i := 1; i < len(match); i++ {
			if strings.TrimSpace(match[i]) != "" {
				cachedVersion = strings.TrimSpace(match[i])
				return cachedVersion, nil
			}
		}

		return "", ctx.Skip("Core.SchemaVersion annotation value could not be extracted")
	}

	suite.AddTest(
		"test_metadata_wildcard_schemaversion",
		"$metadata accepts $schemaversion=*",
		func(ctx *framework.TestContext) error {
			_, err := getSchemaVersion(ctx)
			if err != nil {
				return err
			}

			resp, err := ctx.GET("/$metadata?$schemaversion=*")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("$metadata?$schemaversion=* must succeed when schema versioning is advertised: %v", err))
			}

			return ctx.AssertBodyContains(resp, "Core.SchemaVersion")
		},
	)

	suite.AddTest(
		"test_data_request_with_current_schemaversion",
		"Data request succeeds with advertised $schemaversion value",
		func(ctx *framework.TestContext) error {
			version, err := getSchemaVersion(ctx)
			if err != nil {
				return err
			}

			resp, err := ctx.GET(fmt.Sprintf("/Products?$schemaversion=%s&$top=1", version))
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("request with advertised $schemaversion should succeed: %v", err))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_missing_schemaversion_returns_404",
		"Unknown $schemaversion value returns 404 Not Found",
		func(ctx *framework.TestContext) error {
			_, err := getSchemaVersion(ctx)
			if err != nil {
				return err
			}

			resp, err := ctx.GET("/Products?$schemaversion=__nonexistent_schema_version__&$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 404); err != nil {
				return framework.NewError(fmt.Sprintf("unknown $schemaversion should return 404: %v", err))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_schemaversion_version_negotiation_4_01_vs_4_0",
		"$schemaversion is accepted with OData-MaxVersion 4.01 and rejected when negotiated to 4.0",
		func(ctx *framework.TestContext) error {
			version, err := getSchemaVersion(ctx)
			if err != nil {
				return err
			}

			query := fmt.Sprintf("/Products?$schemaversion=%s&$top=1", version)

			v401Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			v401Resp, err := ctx.GET(query, v401Headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(v401Resp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("4.01 negotiated $schemaversion request should succeed: %v", err))
			}

			v40Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}
			v40Resp, err := ctx.GET(query, v40Headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(v40Resp, http.StatusBadRequest); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 negotiated request must reject 4.01 $schemaversion behavior: %v", err))
			}
			if err := ctx.AssertODataError(v40Resp, http.StatusBadRequest, "not supported in OData 4.0"); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 negotiated $schemaversion rejection must include strict OData error payload: %v", err))
			}

			return nil
		},
	)

	return suite
}
