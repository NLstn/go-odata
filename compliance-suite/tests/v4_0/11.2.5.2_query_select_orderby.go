package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QuerySelectOrderby creates the 11.2.5.2 System Query Option $select and $orderby test suite
func QuerySelectOrderby() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.2 System Query Option $select and $orderby",
		"Tests $select and $orderby query options according to OData v4 specification, including property selection, ordering, and their combinations.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionselect",
	)

	// Test 1: Basic $select with single property
	suite.AddTest(
		"test_select_single",
		"$select with single property",
		func(ctx *framework.TestContext) error {
			select_ := url.QueryEscape("Name")
			resp, err := ctx.GET("/Products?$select=" + select_)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Check that value array exists
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) == 0 {
				return fmt.Errorf("response contains no items")
			}

			// Check first item
			item, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			// Verify Name field is present
			if _, ok := item["Name"]; !ok {
				return fmt.Errorf("selected field 'Name' is missing")
			}

			// Verify that fields NOT in $select are not present (except metadata fields and ID)
			for key := range item {
				// Allow metadata fields and ID (required for identification)
				if key == "@odata.context" || key == "@odata.etag" || key == "@odata.id" || key == "ID" || key == "Name" {
					continue
				}
				// Any other field should not be present
				if key == "Description" || key == "Price" || key == "CategoryID" {
					return fmt.Errorf("response contains field '%s' which was not selected", key)
				}
			}

			return nil
		},
	)

	// Test 2: $select with multiple properties
	suite.AddTest(
		"test_select_multiple",
		"$select with multiple properties",
		func(ctx *framework.TestContext) error {
			select_ := url.QueryEscape("Name,Price")
			resp, err := ctx.GET("/Products?$select=" + select_)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Check that value array exists
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) == 0 {
				return fmt.Errorf("response contains no items")
			}

			// Check first item
			item, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			// Verify both Name and Price are present
			if _, ok := item["Name"]; !ok {
				return fmt.Errorf("selected field 'Name' is missing")
			}
			if _, ok := item["Price"]; !ok {
				return fmt.Errorf("selected field 'Price' is missing")
			}

			// Verify that fields NOT in $select are not present (except metadata fields and ID)
			for key := range item {
				// Allow metadata fields, ID, and selected fields
				if key == "@odata.context" || key == "@odata.etag" || key == "@odata.id" || key == "ID" || key == "Name" || key == "Price" {
					continue
				}
				// Any other field should not be present
				if key == "Description" || key == "CategoryID" {
					return fmt.Errorf("response contains field '%s' which was not selected", key)
				}
			}

			return nil
		},
	)

	// Test 3: Basic $orderby ascending
	suite.AddTest(
		"test_orderby_asc",
		"$orderby ascending",
		func(ctx *framework.TestContext) error {
			orderby := url.QueryEscape("Price asc")
			resp, err := ctx.GET("/Products?$orderby=" + orderby)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Check that value array exists
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) < 2 {
				return fmt.Errorf("need at least 2 items to verify ordering")
			}

			// Extract prices and verify ascending order
			var prices []float64
			for i, v := range value {
				item, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d is not an object", i)
				}
				price, ok := item["Price"].(float64)
				if !ok {
					return fmt.Errorf("item %d missing Price field or not a number", i)
				}
				prices = append(prices, price)
			}

			// Verify ascending order
			for i := 1; i < len(prices); i++ {
				if prices[i] < prices[i-1] {
					return fmt.Errorf("results not ordered ascending: found %.2f after %.2f", prices[i], prices[i-1])
				}
			}

			return nil
		},
	)

	// Test 4: $orderby descending
	suite.AddTest(
		"test_orderby_desc",
		"$orderby descending",
		func(ctx *framework.TestContext) error {
			orderby := url.QueryEscape("Price desc")
			resp, err := ctx.GET("/Products?$orderby=" + orderby)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Check that value array exists
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) < 2 {
				return fmt.Errorf("need at least 2 items to verify ordering")
			}

			// Extract prices and verify descending order
			var prices []float64
			for i, v := range value {
				item, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d is not an object", i)
				}
				price, ok := item["Price"].(float64)
				if !ok {
					return fmt.Errorf("item %d missing Price field or not a number", i)
				}
				prices = append(prices, price)
			}

			// Verify descending order
			for i := 1; i < len(prices); i++ {
				if prices[i] > prices[i-1] {
					return fmt.Errorf("results not ordered descending: found %.2f after %.2f", prices[i], prices[i-1])
				}
			}

			return nil
		},
	)

	// Test 5: $orderby with multiple properties
	suite.AddTest(
		"test_orderby_multiple",
		"$orderby with multiple properties",
		func(ctx *framework.TestContext) error {
			orderby := url.QueryEscape("CategoryID,Price desc")
			resp, err := ctx.GET("/Products?$orderby=" + orderby)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Check that value array exists
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) < 2 {
				return fmt.Errorf("need at least 2 items to verify ordering")
			}

			// Extract items and verify multi-level ordering
			type item struct {
				CategoryID string
				Price      float64
			}
			var items []item
			for i, v := range value {
				obj, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d is not an object", i)
				}
				categoryID, ok := obj["CategoryID"].(string)
				if !ok {
					// CategoryID might be null, treat as empty string
					categoryID = ""
				}
				price, ok := obj["Price"].(float64)
				if !ok {
					return fmt.Errorf("item %d missing Price field or not a number", i)
				}
				items = append(items, item{CategoryID: categoryID, Price: price})
			}

			// Verify ordering: first by CategoryID asc, then by Price desc
			for i := 1; i < len(items); i++ {
				prev := items[i-1]
				curr := items[i]

				// Compare CategoryID first
				if curr.CategoryID < prev.CategoryID {
					return fmt.Errorf("results not ordered by CategoryID: found %s after %s", curr.CategoryID, prev.CategoryID)
				}

				// If CategoryID is the same, verify Price descending
				if curr.CategoryID == prev.CategoryID && curr.Price > prev.Price {
					return fmt.Errorf("results not ordered by Price desc within same CategoryID: found %.2f after %.2f", curr.Price, prev.Price)
				}
			}

			return nil
		},
	)

	// Test 6: Combining $select and $orderby
	suite.AddTest(
		"test_select_orderby_combined",
		"Combining $select and $orderby",
		func(ctx *framework.TestContext) error {
			select_ := url.QueryEscape("Name,Price")
			orderby := url.QueryEscape("Price")
			resp, err := ctx.GET("/Products?$select=" + select_ + "&$orderby=" + orderby)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Check that value array exists
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) < 2 {
				return fmt.Errorf("need at least 2 items to verify")
			}

			// Check first item
			item, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			// Verify selected fields are present
			if _, ok := item["Name"]; !ok {
				return fmt.Errorf("selected field 'Name' is missing")
			}
			if _, ok := item["Price"]; !ok {
				return fmt.Errorf("selected field 'Price' is missing")
			}

			// Extract prices and verify ascending order (default when direction not specified)
			var prices []float64
			for i, v := range value {
				obj, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d is not an object", i)
				}
				price, ok := obj["Price"].(float64)
				if !ok {
					return fmt.Errorf("item %d missing Price field or not a number", i)
				}
				prices = append(prices, price)
			}

			// Verify ascending order
			if !sort.Float64sAreSorted(prices) {
				return fmt.Errorf("results not ordered by Price ascending")
			}

			return nil
		},
	)

	return suite
}
