// Package main demonstrates the usage of exported query types and functions from the go-odata library.
//
// This example shows how external packages can now:
// 1. Use Apply transformation types for programmatic query building
// 2. Access filter operator constants for type-safe filter construction
// 3. Parse filter strings using the public ParseFilter function
package main

import (
	"fmt"
	"log"

	odata "github.com/nlstn/go-odata"
)

func main() {
	fmt.Println("=== go-odata Query Types Export Demo ===")
	fmt.Println()

	// Example 1: Using ParseFilter to parse filter expressions
	demonstrateParseFilter()

	// Example 2: Using filter operator constants
	demonstrateOperatorConstants()

	// Example 3: Building Apply transformations programmatically
	demonstrateApplyTransformations()

	// Example 4: Using ParserConfig
	demonstrateParserConfig()
}

func demonstrateParseFilter() {
	fmt.Println("1. Parsing Filter Expressions")
	fmt.Println("------------------------------")

	// Parse a simple equality filter
	filter, err := odata.ParseFilter("Name eq 'John'")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed filter:\n")
	fmt.Printf("  Property: %s\n", filter.Property)
	fmt.Printf("  Operator: %s\n", filter.Operator)
	fmt.Printf("  Value: %v\n", filter.Value)

	// Parse a complex filter with logical operators
	complexFilter, err := odata.ParseFilter("Age gt 18 and Status eq 'Active'")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nComplex filter:\n")
	fmt.Printf("  Logical: %s\n", complexFilter.Logical)
	fmt.Printf("  Left - Property: %s, Operator: %s, Value: %v\n",
		complexFilter.Left.Property, complexFilter.Left.Operator, complexFilter.Left.Value)
	fmt.Printf("  Right - Property: %s, Operator: %s, Value: %v\n",
		complexFilter.Right.Property, complexFilter.Right.Operator, complexFilter.Right.Value)
	fmt.Println()
}

func demonstrateOperatorConstants() {
	fmt.Println("2. Using Filter Operator Constants")
	fmt.Println("-----------------------------------")

	// Users can now reference operator constants symbolically
	fmt.Printf("Equal operator: %s\n", odata.OpEqual)
	fmt.Printf("Greater than operator: %s\n", odata.OpGreaterThan)
	fmt.Printf("Contains operator: %s\n", odata.OpContains)
	fmt.Printf("Logical AND: %s\n", odata.LogicalAnd)
	fmt.Printf("Logical OR: %s\n", odata.LogicalOr)

	// This is useful for building or inspecting filter expressions programmatically
	filter, err := odata.ParseFilter("Name eq 'John'")
	if err != nil {
		log.Fatal(err)
	}
	if filter.Operator == odata.OpEqual {
		fmt.Println("\nFilter uses equality operator ✓")
	}
	fmt.Println()
}

func demonstrateApplyTransformations() {
	fmt.Println("3. Building Apply Transformations")
	fmt.Println("---------------------------------")

	// Users implementing QueryOptions overwrites can now properly type their code
	queryOptions := &odata.QueryOptions{
		Apply: []odata.ApplyTransformation{
			{
				Type: odata.ApplyTypeGroupBy,
				GroupBy: &odata.GroupByTransformation{
					Properties: []string{"Category", "Region"},
					Transform: []odata.ApplyTransformation{
						{
							Type: odata.ApplyTypeAggregate,
							Aggregate: &odata.AggregateTransformation{
								Expressions: []odata.AggregateExpression{
									{
										Property: "Price",
										Method:   odata.AggregationSum,
										Alias:    "TotalPrice",
									},
									{
										Property: "Quantity",
										Method:   odata.AggregationAvg,
										Alias:    "AvgQuantity",
									},
								},
							},
						},
					},
				},
			},
			{
				Type: odata.ApplyTypeCompute,
				Compute: &odata.ComputeTransformation{
					Expressions: []odata.ComputeExpression{
						{
							Alias: "Revenue",
							// Expression would be set here
						},
					},
				},
			},
		},
	}

	fmt.Printf("Created QueryOptions with %d Apply transformations:\n", len(queryOptions.Apply))
	for i, transform := range queryOptions.Apply {
		fmt.Printf("  %d. Type: %s\n", i+1, transform.Type)
		if transform.GroupBy != nil {
			fmt.Printf("     - GroupBy properties: %v\n", transform.GroupBy.Properties)
			fmt.Printf("     - Nested transformations: %d\n", len(transform.GroupBy.Transform))
			if len(transform.GroupBy.Transform) > 0 && transform.GroupBy.Transform[0].Aggregate != nil {
				fmt.Printf("     - Aggregate expressions: %d\n", len(transform.GroupBy.Transform[0].Aggregate.Expressions))
				for j, expr := range transform.GroupBy.Transform[0].Aggregate.Expressions {
					fmt.Printf("       %d. %s(%s) as %s\n", j+1, expr.Method, expr.Property, expr.Alias)
				}
			}
		}
		if transform.Compute != nil {
			fmt.Printf("     - Compute expressions: %d\n", len(transform.Compute.Expressions))
		}
	}
	fmt.Println()
}

func demonstrateParserConfig() {
	fmt.Println("4. Using ParserConfig")
	fmt.Println("---------------------")

	// Users can now configure parser behavior
	config := &odata.ParserConfig{
		MaxInClauseSize: 100,
		MaxExpandDepth:  5,
	}

	fmt.Printf("Parser configuration:\n")
	fmt.Printf("  MaxInClauseSize: %d\n", config.MaxInClauseSize)
	fmt.Printf("  MaxExpandDepth: %d\n", config.MaxExpandDepth)
	fmt.Println()
}
