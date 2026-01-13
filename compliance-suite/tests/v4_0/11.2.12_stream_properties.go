package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// StreamProperties creates the 11.2.12 Stream Properties and Media Entities test suite
func StreamProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.12 Stream Properties and Media Entities",
		"Tests media entities, stream properties, and $value for streams according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_MediaEntities",
	)

	// Note: Stream properties and media entities are advanced OData features
	// Many implementations may not support them initially

	getMediaPath := func(ctx *framework.TestContext, index int) (string, error) {
		ids, err := fetchEntityIDs(ctx, "MediaItems", index+1)
		if err != nil {
			return "", err
		}
		if len(ids) <= index {
			return "", fmt.Errorf("need at least %d media item(s)", index+1)
		}
		return fmt.Sprintf("/MediaItems(%s)", ids[index]), nil
	}

	// Helper function to get product path for each test
	// Note: Must refetch on each call because database is reseeded between tests
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		return firstEntityPath(ctx, "Products")
	}

	// Test 1: Request media entity
	suite.AddTest(
		"test_media_entity",
		"Request media entity (optional)",
		func(ctx *framework.TestContext) error {
			mediaPath, err := getMediaPath(ctx, 0)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(mediaPath)
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError("media entities not implemented")
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 2: Request media entity $value (binary content)
	suite.AddTest(
		"test_media_entity_value",
		"Request media entity binary content",
		func(ctx *framework.TestContext) error {
			mediaPath, err := getMediaPath(ctx, 0)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(mediaPath + "/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError("media entity $value not implemented")
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 3: Request stream property
	suite.AddTest(
		"test_stream_property",
		"Request stream property (optional)",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "/Photo")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError("stream properties not implemented")
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 4: Media entity with content type
	suite.AddTest(
		"test_media_content_type",
		"Media entity Content-Type header",
		func(ctx *framework.TestContext) error {
			mediaPath, err := getMediaPath(ctx, 0)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(mediaPath + "/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				// Check for appropriate Content-Type (image/*, application/*, etc.)
				contentType := resp.Headers.Get("Content-Type")
				if contentType == "" {
					return fmt.Errorf("specification violation - missing Content-Type header")
				}
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError(fmt.Sprintf("media content unavailable (status: %d)", resp.StatusCode))
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 5: POST media entity (upload)
	suite.AddTest(
		"test_post_media_entity",
		"Create media entity (upload)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw("/MediaItems", []byte("fake-binary-data"), "image/png")
			if err != nil {
				return err
			}

			if resp.StatusCode == 201 || resp.StatusCode == 200 || resp.StatusCode == 204 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError(fmt.Sprintf("media entity creation unsupported (status: %d)", resp.StatusCode))
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 6: PUT to update media entity content
	suite.AddTest(
		"test_put_media_value",
		"Update media entity content",
		func(ctx *framework.TestContext) error {
			mediaPath, err := getMediaPath(ctx, 0)
			if err != nil {
				return err
			}
			resp, err := ctx.PUTRaw(mediaPath+"/$value", []byte("updated-binary-data"), "image/png")
			if err != nil {
				return err
			}

			if resp.StatusCode == 204 || resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError(fmt.Sprintf("media entity update unsupported (status: %d)", resp.StatusCode))
			}

			return fmt.Errorf("unexpected status %d updating media entity. Response: %s", resp.StatusCode, string(resp.Body))
		},
	)

	// Test 7: Media entity metadata
	suite.AddTest(
		"test_media_metadata",
		"Access media entity metadata",
		func(ctx *framework.TestContext) error {
			mediaPath, err := getMediaPath(ctx, 0)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(mediaPath)
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError(fmt.Sprintf("media entity metadata missing (status: %d)", resp.StatusCode))
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 8: Stream property in metadata
	suite.AddTest(
		"test_stream_in_metadata",
		"Stream properties in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("specification violation - metadata endpoint unavailable (status: %d)", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if strings.Contains(bodyStr, "HasStream") || strings.Contains(bodyStr, "Stream") {
				return nil
			}

			return fmt.Errorf("specification violation - metadata missing stream annotations")
		},
	)

	// Test 9: Accept header for media content
	suite.AddTest(
		"test_media_accept_header",
		"Accept header for media content",
		func(ctx *framework.TestContext) error {
			mediaPath, err := getMediaPath(ctx, 0)
			if err != nil {
				return err
			}
			resp, err := ctx.GETWithHeaders(mediaPath+"/$value", map[string]string{
				"Accept": "image/png",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError(fmt.Sprintf("media stream negotiation failed (status: %d)", resp.StatusCode))
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 10: DELETE media entity
	suite.AddTest(
		"test_delete_media",
		"Delete media entity",
		func(ctx *framework.TestContext) error {
			mediaPath, err := getMediaPath(ctx, 1)
			if err != nil {
				return err
			}
			resp, err := ctx.DELETE(mediaPath)
			if err != nil {
				return err
			}

			if resp.StatusCode == 204 || resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError(fmt.Sprintf("media entity deletion failed (status: %d)", resp.StatusCode))
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 11: Media link entry
	suite.AddTest(
		"test_media_link_entry",
		"Media link entry annotations",
		func(ctx *framework.TestContext) error {
			mediaPath, err := getMediaPath(ctx, 0)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(mediaPath)
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				bodyStr := string(resp.Body)
				if strings.Contains(bodyStr, "@odata.media") {
					return nil
				}
				return fmt.Errorf("specification violation - missing media link annotations")
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError(fmt.Sprintf("media entity endpoint unavailable (status: %d)", resp.StatusCode))
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 12: Stream property $value
	suite.AddTest(
		"test_stream_property_value",
		"Stream property $value access",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "/Photo/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
				return framework.NewError(fmt.Sprintf("stream property $value missing (status: %d)", resp.StatusCode))
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	return suite
}
