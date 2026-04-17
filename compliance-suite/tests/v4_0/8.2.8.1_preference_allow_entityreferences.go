package v4_0

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// PreferenceAllowEntityReferences creates the 8.2.8.1 odata.allow-entityreferences test suite.
func PreferenceAllowEntityReferences() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8.1 Preference odata.allow-entityreferences",
		"Validates that the odata.allow-entityreferences Prefer option is accepted, honored, and reported via Preference-Applied.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderPrefer",
	)

	suite.AddTest(
		"test_allow_entityreferences_accepted_on_collection",
		"Prefer: odata.allow-entityreferences is accepted on collection reads",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3",
				framework.Header{Key: "Prefer", Value: "odata.allow-entityreferences"})
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
		"test_allow_entityreferences_preference_applied_echoed",
		"Preference-Applied echoes odata.allow-entityreferences when honored",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3",
				framework.Header{Key: "Prefer", Value: "odata.allow-entityreferences"})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx, got %d", resp.StatusCode))
			}

			applied := resp.Headers.Get("Preference-Applied")
			if applied == "" {
				// Server may choose not to honor the preference — that is spec-compliant.
				return nil
			}

			// When Preference-Applied is returned it MUST contain odata.allow-entityreferences.
			if !strings.Contains(applied, "odata.allow-entityreferences") {
				return framework.NewError(fmt.Sprintf(
					"Preference-Applied must contain odata.allow-entityreferences when honored, got %q", applied))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_allow_entityreferences_accepted_on_entity_read",
		"Prefer: odata.allow-entityreferences is accepted on individual entity reads",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products(1)",
				framework.Header{Key: "Prefer", Value: "odata.allow-entityreferences"})
			if err != nil {
				return err
			}

			// 404 is acceptable when no product with ID 1 exists; we just need the server to not error on the preference.
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
				return framework.NewError(fmt.Sprintf("expected 200 or 404, got %d", resp.StatusCode))
			}

			return nil
		},
	)

	return suite
}
