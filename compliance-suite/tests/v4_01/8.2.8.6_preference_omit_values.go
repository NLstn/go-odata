package v4_01

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// PreferenceOmitValues creates the 8.2.8.6 omit-values preference test suite.
func PreferenceOmitValues() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8.6 Preference omit-values",
		"Validates OData 4.01 omit-values preference behavior and compatibility with the OData 4.0 prefixed form.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_Preferenceomitvalues",
	)

	suite.AddTest(
		"test_omit_values_unprefixed_is_accepted",
		"Prefer: omit-values=nulls is accepted on data requests",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3", framework.Header{Key: "Prefer", Value: "omit-values=nulls"})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx for omit-values request, got %d", resp.StatusCode))
			}

			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return framework.NewError(fmt.Sprintf("response must remain a valid collection payload: %v", err))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_omit_values_prefixed_alias_is_accepted",
		"Prefer: odata.omit-values=nulls is accepted for 4.0 compatibility",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3", framework.Header{Key: "Prefer", Value: "odata.omit-values=nulls"})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx for odata.omit-values request, got %d", resp.StatusCode))
			}

			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return framework.NewError(fmt.Sprintf("response must remain a valid collection payload: %v", err))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_omit_values_preference_applied_is_consistent",
		"If Preference-Applied is returned, it echoes the applied omit-values form",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3", framework.Header{Key: "Prefer", Value: "omit-values=defaults"})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx for omit-values=defaults request, got %d", resp.StatusCode))
			}

			applied := resp.Headers.Get("Preference-Applied")
			if applied == "" {
				return nil
			}

			if !strings.Contains(applied, "omit-values=defaults") && !strings.Contains(applied, "odata.omit-values=defaults") {
				return framework.NewError(fmt.Sprintf("unexpected Preference-Applied value for omit-values: %q", applied))
			}

			return nil
		},
	)

	return suite
}
