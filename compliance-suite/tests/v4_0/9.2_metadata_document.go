package v4_0

import (
	"bytes"
	"encoding/xml"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// MetadataDocument creates the 9.2 Metadata Document test suite
func MetadataDocument() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"9.2 Metadata Document",
		"Tests metadata document structure and format, including XML validity, required elements, and Content-Type headers according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_MetadataDocumentRequest",
	)

	// Test 1: Metadata document is accessible at $metadata
	suite.AddTest(
		"test_metadata_accessible",
		"Metadata document accessible at $metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Metadata Content-Type is application/xml
	suite.AddTest(
		"test_metadata_content_type",
		"Metadata Content-Type is application/xml",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(contentType, "application/xml") {
				return framework.NewError("Metadata Content-Type must be application/xml")
			}

			return nil
		},
	)

	// Test 3: Metadata contains Edmx element
	suite.AddTest(
		"test_metadata_edmx_element",
		"Metadata contains Edmx root element",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<edmx:Edmx") && !strings.Contains(body, "<Edmx") {
				return framework.NewError("Metadata must contain Edmx root element")
			}

			return nil
		},
	)

	// Test 4: Metadata contains DataServices element
	suite.AddTest(
		"test_metadata_dataservices_element",
		"Metadata contains DataServices element",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<edmx:DataServices") && !strings.Contains(body, "<DataServices") {
				return framework.NewError("Metadata must contain DataServices element")
			}

			return nil
		},
	)

	// Test 5: Metadata contains Schema element
	suite.AddTest(
		"test_metadata_schema_element",
		"Metadata contains Schema element",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Schema") {
				return framework.NewError("Metadata must contain Schema element")
			}

			return nil
		},
	)

	// Test 6: Metadata contains EntityType definitions
	suite.AddTest(
		"test_metadata_entitytype",
		"Metadata contains EntityType definitions",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<EntityType") {
				return framework.NewError("Metadata must contain EntityType definitions")
			}

			return nil
		},
	)

	// Test 7: Metadata contains EntityContainer
	suite.AddTest(
		"test_metadata_entitycontainer",
		"Metadata contains EntityContainer",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<EntityContainer") {
				return framework.NewError("Metadata must contain EntityContainer")
			}

			return nil
		},
	)

	// Test 8: Metadata is valid XML
	suite.AddTest(
		"test_metadata_valid_xml",
		"Metadata document is valid XML",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			// Use Go's xml.Decoder to validate XML
			decoder := xml.NewDecoder(bytes.NewReader(resp.Body))
			for {
				_, err := decoder.Token()
				if err != nil {
					if err.Error() == "EOF" {
						// Successfully parsed entire document
						break
					}
					return framework.NewError("Metadata document is not valid XML: " + err.Error())
				}
			}

			return nil
		},
	)

	return suite
}
