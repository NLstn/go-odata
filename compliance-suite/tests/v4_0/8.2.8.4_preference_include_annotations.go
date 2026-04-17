package v4_0

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// PreferenceIncludeAnnotations creates the 8.2.8.4 odata.include-annotations test suite.
func PreferenceIncludeAnnotations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8.4 Preference odata.include-annotations",
		"Validates that the odata.include-annotations Prefer option is accepted, applied, and reported via Preference-Applied.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderPrefer",
	)

	suite.AddTest(
		"test_include_annotations_wildcard_accepted",
		"Prefer: odata.include-annotations=\"*\" is accepted on collection reads",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3",
				framework.Header{Key: "Prefer", Value: `odata.include-annotations="*"`})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx, got %d", resp.StatusCode))
			}

			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return framework.NewError(fmt.Sprintf("response must be a valid collection payload: %v", err))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_include_annotations_preference_applied_echoed",
		"Preference-Applied echoes odata.include-annotations when honored",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3",
				framework.Header{Key: "Prefer", Value: `odata.include-annotations="*"`})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx, got %d", resp.StatusCode))
			}

			applied := resp.Headers.Get("Preference-Applied")
			if applied == "" {
				return framework.NewError("Preference-Applied header must be set when odata.include-annotations is honored")
			}

			if !strings.Contains(applied, "odata.include-annotations") {
				return framework.NewError(fmt.Sprintf(
					"Preference-Applied must contain odata.include-annotations, got %q", applied))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_include_annotations_exclude_all_accepted",
		"Prefer: odata.include-annotations=\"-*\" is accepted (suppress all annotations)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3",
				framework.Header{Key: "Prefer", Value: `odata.include-annotations="-*"`})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx, got %d", resp.StatusCode))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_include_annotations_specific_term_accepted",
		"Prefer: odata.include-annotations with a specific term is accepted",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3",
				framework.Header{Key: "Prefer", Value: `odata.include-annotations="Org.OData.Core.V1.Computed"`})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx, got %d", resp.StatusCode))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_include_annotations_combined_rules_accepted",
		"Prefer: odata.include-annotations with combined include/exclude rules is accepted",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3",
				framework.Header{Key: "Prefer", Value: `odata.include-annotations="*,-Org.OData.Core.V1.Description"`})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf(
					"expected 2xx for combined include-annotations rule, got %d", resp.StatusCode))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_include_annotations_on_entity_read",
		"Prefer: odata.include-annotations is accepted on individual entity reads",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products(1)",
				framework.Header{Key: "Prefer", Value: `odata.include-annotations="*"`})
			if err != nil {
				return err
			}

			// 404 is acceptable when no product with ID 1 exists; we just want the server to handle the preference.
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
				return framework.NewError(fmt.Sprintf("expected 200 or 404, got %d", resp.StatusCode))
			}

			if resp.StatusCode == http.StatusOK {
				applied := resp.Headers.Get("Preference-Applied")
				if applied == "" {
					return framework.NewError("Preference-Applied must be set on entity read with include-annotations")
				}
				if !strings.Contains(applied, "odata.include-annotations") {
					return framework.NewError(fmt.Sprintf(
						"Preference-Applied must contain odata.include-annotations, got %q", applied))
				}
			}

			return nil
		},
	)

	return suite
}
