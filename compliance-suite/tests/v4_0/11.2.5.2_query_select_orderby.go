package v4_0

import (
	"fmt"
	"net/url"

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
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}

			item := items[0]

			// Verify Name field is present
			if err := ctx.AssertEntityHasFields(item, "Name"); err != nil {
				return err
			}

			// Verify that fields NOT in $select are not present.
			if err := ctx.AssertEntityOnlyAllowedFields(item, "@odata.context", "@odata.etag", "@odata.id", "ID", "Name"); err != nil {
				return err
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
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}

			item := items[0]

			// Verify both Name and Price are present
			if err := ctx.AssertEntityHasFields(item, "Name", "Price"); err != nil {
				return err
			}

			// Verify that fields NOT in $select are not present.
			if err := ctx.AssertEntityOnlyAllowedFields(item, "@odata.context", "@odata.etag", "@odata.id", "ID", "Name", "Price"); err != nil {
				return err
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
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 2); err != nil {
				return err
			}

			return ctx.AssertEntitiesSortedByFloat(items, "Price", true)
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
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 2); err != nil {
				return err
			}

			return ctx.AssertEntitiesSortedByFloat(items, "Price", false)
		},
	)

	// Test 5: $orderby with multiple properties
	suite.AddTest(
		"test_orderby_multiple",
		"$orderby with multiple properties",
		func(ctx *framework.TestContext) error {
			orderby := url.QueryEscape("Name,Price desc")
			resp, err := ctx.GET("/Products?$orderby=" + orderby)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 2); err != nil {
				return err
			}

			// Extract items and verify multi-level ordering
			type item struct {
				Name  string
				Price float64
			}
			var extracted []item
			for i, obj := range items {
				name, ok := obj["Name"].(string)
				if !ok {
					return fmt.Errorf("item %d missing Name field or not a string", i)
				}
				price, ok := obj["Price"].(float64)
				if !ok {
					return fmt.Errorf("item %d missing Price field or not a number", i)
				}
				extracted = append(extracted, item{Name: name, Price: price})
			}

			// Verify ordering: first by Name asc, then by Price desc
			for i := 1; i < len(extracted); i++ {
				prev := extracted[i-1]
				curr := extracted[i]

				// Compare Name first
				if curr.Name < prev.Name {
					return fmt.Errorf("results not ordered by Name: found %s after %s", curr.Name, prev.Name)
				}

				// If Name is the same, verify Price descending
				if curr.Name == prev.Name && curr.Price > prev.Price {
					return fmt.Errorf("results not ordered by Price desc within same Name: found %.2f after %.2f", curr.Price, prev.Price)
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
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 2); err != nil {
				return err
			}

			// Check first item
			item := items[0]

			// Verify selected fields are present
			if err := ctx.AssertEntityHasFields(item, "Name", "Price"); err != nil {
				return err
			}
			if err := ctx.AssertEntityOnlyAllowedFields(item, "@odata.context", "@odata.etag", "@odata.id", "ID", "Name", "Price"); err != nil {
				return err
			}

			return ctx.AssertEntitiesSortedByFloat(items, "Price", true)
		},
	)

	return suite
}
