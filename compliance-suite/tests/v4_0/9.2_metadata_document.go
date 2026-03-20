package v4_0

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

const (
	edmxNamespace = "http://docs.oasis-open.org/odata/ns/edmx"
	edmNamespace  = "http://docs.oasis-open.org/odata/ns/edm"
)

type csdlMetadataDocument struct {
	XMLName      xml.Name                 `xml:"Edmx"`
	Version      string                   `xml:"Version,attr"`
	DataServices csdlMetadataDataServices `xml:"DataServices"`
}

type csdlMetadataDataServices struct {
	XMLName xml.Name             `xml:"DataServices"`
	Schemas []csdlMetadataSchema `xml:"Schema"`
}

type csdlMetadataSchema struct {
	XMLName         xml.Name                     `xml:"Schema"`
	Namespace       string                       `xml:"Namespace,attr"`
	EntityTypes     []csdlMetadataEntityType     `xml:"EntityType"`
	EntityContainer *csdlMetadataEntityContainer `xml:"EntityContainer"`
}

type csdlMetadataEntityType struct {
	Name string           `xml:"Name,attr"`
	Key  *csdlMetadataKey `xml:"Key"`
}

type csdlMetadataKey struct {
	PropertyRefs []csdlMetadataPropertyRef `xml:"PropertyRef"`
}

type csdlMetadataPropertyRef struct {
	Name string `xml:"Name,attr"`
}

type csdlMetadataEntityContainer struct {
	Name       string                  `xml:"Name,attr"`
	EntitySets []csdlMetadataEntitySet `xml:"EntitySet"`
	Singletons []csdlMetadataSingleton `xml:"Singleton"`
}

type csdlMetadataEntitySet struct {
	Name       string `xml:"Name,attr"`
	EntityType string `xml:"EntityType,attr"`
}

type csdlMetadataSingleton struct {
	Name string `xml:"Name,attr"`
	Type string `xml:"Type,attr"`
}

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
		"Metadata contains Edmx root element with the expected namespace",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}

			if metadata.XMLName.Local != "Edmx" {
				return framework.NewError(fmt.Sprintf("metadata root element must be Edmx, got %s", metadata.XMLName.Local))
			}
			if metadata.XMLName.Space != edmxNamespace {
				return framework.NewError(fmt.Sprintf("metadata root namespace must be %s, got %s", edmxNamespace, metadata.XMLName.Space))
			}
			if strings.TrimSpace(metadata.Version) == "" {
				return framework.NewError("metadata document missing edmx:Edmx Version attribute")
			}

			return nil
		},
	)

	// Test 4: Metadata contains DataServices element
	suite.AddTest(
		"test_metadata_dataservices_element",
		"Metadata contains DataServices element in the Edmx namespace",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}

			if metadata.DataServices.XMLName.Local != "DataServices" {
				return framework.NewError("metadata must contain DataServices element")
			}
			if metadata.DataServices.XMLName.Space != edmxNamespace {
				return framework.NewError(fmt.Sprintf("DataServices namespace must be %s, got %s", edmxNamespace, metadata.DataServices.XMLName.Space))
			}
			if len(metadata.DataServices.Schemas) == 0 {
				return framework.NewError("metadata DataServices element must contain at least one Schema")
			}

			return nil
		},
	)

	// Test 5: Metadata contains Schema element
	suite.AddTest(
		"test_metadata_schema_element",
		"Metadata contains Schema elements with namespaces",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}

			for i, schema := range metadata.DataServices.Schemas {
				if schema.XMLName.Local != "Schema" {
					return framework.NewError(fmt.Sprintf("schema %d has unexpected element name %s", i, schema.XMLName.Local))
				}
				if schema.XMLName.Space != edmNamespace {
					return framework.NewError(fmt.Sprintf("schema %d namespace must be %s, got %s", i, edmNamespace, schema.XMLName.Space))
				}
				if strings.TrimSpace(schema.Namespace) == "" {
					return framework.NewError(fmt.Sprintf("schema %d missing Namespace attribute", i))
				}
			}

			return nil
		},
	)

	// Test 6: Metadata contains EntityType definitions
	suite.AddTest(
		"test_metadata_entitytype",
		"Metadata contains EntityType definitions with key properties",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}

			foundEntityType := false
			for _, schema := range metadata.DataServices.Schemas {
				for _, entityType := range schema.EntityTypes {
					foundEntityType = true
					if strings.TrimSpace(entityType.Name) == "" {
						return framework.NewError("EntityType definition missing Name attribute")
					}
					if entityType.Key == nil || len(entityType.Key.PropertyRefs) == 0 {
						return framework.NewError(fmt.Sprintf("EntityType %s missing Key/PropertyRef definition", entityType.Name))
					}
				}
			}
			if !foundEntityType {
				return framework.NewError("metadata must contain at least one EntityType definition")
			}

			return nil
		},
	)

	// Test 7: Metadata contains EntityContainer
	suite.AddTest(
		"test_metadata_entitycontainer",
		"Metadata contains EntityContainer with advertised resources",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}

			foundContainer := false
			for _, schema := range metadata.DataServices.Schemas {
				if schema.EntityContainer == nil {
					continue
				}
				foundContainer = true
				if strings.TrimSpace(schema.EntityContainer.Name) == "" {
					return framework.NewError("EntityContainer missing Name attribute")
				}
				if len(schema.EntityContainer.EntitySets) == 0 && len(schema.EntityContainer.Singletons) == 0 {
					return framework.NewError("EntityContainer must advertise at least one EntitySet or Singleton")
				}
			}
			if !foundContainer {
				return framework.NewError("metadata must contain EntityContainer")
			}

			return nil
		},
	)

	// Test 8: Metadata is valid XML
	suite.AddTest(
		"test_metadata_valid_xml",
		"Metadata document is valid XML and can be unmarshaled as CSDL",
		func(ctx *framework.TestContext) error {
			_, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}

			return nil
		},
	)

	return suite
}

func loadMetadataDocument(ctx *framework.TestContext) (*csdlMetadataDocument, *framework.HTTPResponse, error) {
	resp, err := ctx.GET("/$metadata")
	if err != nil {
		return nil, nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, resp, err
	}

	var metadata csdlMetadataDocument
	if err := xml.Unmarshal(resp.Body, &metadata); err != nil {
		return nil, resp, framework.NewError("Metadata document is not valid XML: " + err.Error())
	}

	return &metadata, resp, nil
}
