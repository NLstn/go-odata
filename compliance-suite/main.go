package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nlstn/go-odata/compliance-suite/framework"
	v4_0 "github.com/nlstn/go-odata/compliance-suite/tests/v4_0"
)

var (
	serverURL      = flag.String("server", "http://localhost:9090", "OData server URL")
	dbType         = flag.String("db", "sqlite", "Database type (sqlite or postgres)")
	dbDSN          = flag.String("dsn", "", "Database DSN/connection string")
	version        = flag.String("version", "all", "OData version to test (4.0, 4.01, or all)")
	pattern        = flag.String("pattern", "", "Run only tests matching pattern")
	debug          = flag.Bool("debug", false, "Enable debug mode with full HTTP details")
	externalServer = flag.Bool("external-server", false, "Use an external server (don't start/stop)")
	outputFile     = flag.String("output", "compliance-report.md", "Output file for the report")
)

type TestSuiteInfo struct {
	Name    string
	Version string
	Suite   func() *framework.TestSuite
}

func main() {
	flag.Parse()

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║     OData v4 Compliance Test Suite                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Server URL: %s\n", *serverURL)
	fmt.Printf("Database:   %s", *dbType)
	if *dbDSN != "" {
		fmt.Print(" (dsn provided)")
	}
	fmt.Println()
	fmt.Printf("Version:    %s\n", *version)
	fmt.Printf("Report File: %s\n", *outputFile)
	if *debug {
		fmt.Println("Debug Mode: ENABLED")
	}
	fmt.Println()

	// Start compliance server if not using external server
	var serverCmd *exec.Cmd
	if !*externalServer {
		var err error
		serverCmd, err = startComplianceServer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start compliance server: %v\n", err)
			os.Exit(1)
		}
		defer stopComplianceServer(serverCmd)
	} else {
		if !checkServerConnectivity() {
			fmt.Fprintln(os.Stderr, "Error: Cannot connect to external server")
			os.Exit(1)
		}
	}

	// Gather test suites
	testSuites := []TestSuiteInfo{}

	// Register v4.0 tests
	if *version == "all" || *version == "4.0" {
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "1.1_introduction",
			Version: "4.0",
			Suite:   v4_0.Introduction,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "2.1_conformance",
			Version: "4.0",
			Suite:   v4_0.Conformance,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.1_edmx_element",
			Version: "4.0",
			Suite:   v4_0.EDMXElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.2_dataservices_element",
			Version: "4.0",
			Suite:   v4_0.DataServicesElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.3_reference_element",
			Version: "4.0",
			Suite:   v4_0.ReferenceElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.4_include_element",
			Version: "4.0",
			Suite:   v4_0.IncludeElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.5_includeannotations_element",
			Version: "4.0",
			Suite:   v4_0.IncludeAnnotationsElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.1_nominal_types",
			Version: "4.0",
			Suite:   v4_0.NominalTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.2_structured_types",
			Version: "4.0",
			Suite:   v4_0.StructuredTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.3_navigation_properties",
			Version: "4.0",
			Suite:   v4_0.NavigationProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.4_primitive_types",
			Version: "4.0",
			Suite:   v4_0.PrimitiveTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.5_builtin_abstract_types",
			Version: "4.0",
			Suite:   v4_0.BuiltInAbstractTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.6_annotations",
			Version: "4.0",
			Suite:   v4_0.MetadataAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1_primitive_data_types",
			Version: "4.0",
			Suite:   v4_0.PrimitiveDataTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.1_numeric_edge_cases",
			Version: "4.0",
			Suite:   v4_0.NumericEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.2_nullable_properties",
			Version: "4.0",
			Suite:   v4_0.NullableProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.3_collection_properties",
			Version: "4.0",
			Suite:   v4_0.CollectionProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.4_temporal_data_types",
			Version: "4.0",
			Suite:   v4_0.TemporalDataTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2_complex_types",
			Version: "4.0",
			Suite:   v4_0.ComplexTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2.1_complex_filter",
			Version: "4.0",
			Suite:   v4_0.ComplexFilter,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2.2_complex_orderby",
			Version: "4.0",
			Suite:   v4_0.ComplexOrderBy,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.3_enum_types",
			Version: "4.0",
			Suite:   v4_0.EnumTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.3_enum_metadata_members",
			Version: "4.0",
			Suite:   v4_0.EnumMetadataMembers,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.4_type_definitions",
			Version: "4.0",
			Suite:   v4_0.TypeDefinitions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "6.1_extensibility",
			Version: "4.0",
			Suite:   v4_0.Extensibility,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "7.1.1_unicode_strings",
			Version: "4.0",
			Suite:   v4_0.UnicodeStrings,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.1_header_content_type",
			Version: "4.0",
			Suite:   v4_0.HeaderContentType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.2_request_headers",
			Version: "4.0",
			Suite:   v4_0.RequestHeaders,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.3_response_headers",
			Version: "4.0",
			Suite:   v4_0.ResponseHeaders,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.5_response_status_codes",
			Version: "4.0",
			Suite:   v4_0.ResponseStatusCodes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.1_cache_control_header",
			Version: "4.0",
			Suite:   v4_0.CacheControlHeader,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.2_header_if_match",
			Version: "4.0",
			Suite:   v4_0.HeaderIfMatch,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.3_header_odata_entityid",
			Version: "4.0",
			Suite:   v4_0.HeaderODataEntityId,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.4_header_content_id",
			Version: "4.0",
			Suite:   v4_0.HeaderContentId,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.5_header_location",
			Version: "4.0",
			Suite:   v4_0.HeaderLocation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.6_header_odata_version",
			Version: "4.0",
			Suite:   v4_0.HeaderODataVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.7_header_accept",
			Version: "4.0",
			Suite:   v4_0.HeaderAccept,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8_header_prefer",
			Version: "4.0",
			Suite:   v4_0.HeaderPrefer,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.9_header_maxversion",
			Version: "4.0",
			Suite:   v4_0.HeaderMaxVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.3_error_responses",
			Version: "4.0",
			Suite:   v4_0.ErrorResponses,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.4_error_response_consistency",
			Version: "4.0",
			Suite:   v4_0.ErrorResponseConsistency,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.1_service_document",
			Version: "4.0",
			Suite:   v4_0.ServiceDocument,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.2_metadata_document",
			Version: "4.0",
			Suite:   v4_0.MetadataDocument,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.3_annotations_metadata",
			Version: "4.0",
			Suite:   v4_0.AnnotationsMetadata,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "10.1_json_format",
			Version: "4.0",
			Suite:   v4_0.JSONFormat,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.1_resource_path",
			Version: "4.0",
			Suite:   v4_0.ResourcePath,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.1_addressing_entities",
			Version: "4.0",
			Suite:   v4_0.AddressingEntities,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.2_canonical_url",
			Version: "4.0",
			Suite:   v4_0.CanonicalURL,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.3_property_access",
			Version: "4.0",
			Suite:   v4_0.PropertyAccess,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.4_collection_operations",
			Version: "4.0",
			Suite:   v4_0.CollectionOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.4.1_query_search",
			Version: "4.0",
			Suite:   v4_0.QuerySearch,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.1_query_filter",
			Version: "4.0",
			Suite:   v4_0.QueryFilter,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.2_query_select_orderby",
			Version: "4.0",
			Suite:   v4_0.QuerySelectOrderby,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.3_query_top_skip",
			Version: "4.0",
			Suite:   v4_0.QueryTopSkip,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4_query_apply",
			Version: "4.0",
			Suite:   v4_0.QueryApply,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4.1_advanced_apply",
			Version: "4.0",
			Suite:   v4_0.AdvancedApply,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.5_query_count",
			Version: "4.0",
			Suite:   v4_0.QueryCount,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.6_query_expand",
			Version: "4.0",
			Suite:   v4_0.QueryExpand,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.7_query_skiptoken",
			Version: "4.0",
			Suite:   v4_0.QuerySkiptoken,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.9_nested_expand_options",
			Version: "4.0",
			Suite:   v4_0.NestedExpandOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.10_query_option_combinations",
			Version: "4.0",
			Suite:   v4_0.QueryOptionCombinations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.12_pagination_edge_cases",
			Version: "4.0",
			Suite:   v4_0.PaginationEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.6_query_format",
			Version: "4.0",
			Suite:   v4_0.QueryFormat,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.7_metadata_levels",
			Version: "4.0",
			Suite:   v4_0.MetadataLevels,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.8_delta_links",
			Version: "4.0",
			Suite:   v4_0.DeltaLinks,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.9_lambda_operators",
			Version: "4.0",
			Suite:   v4_0.LambdaOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.10_addressing_operations",
			Version: "4.0",
			Suite:   v4_0.AddressingOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.11_property_value",
			Version: "4.0",
			Suite:   v4_0.PropertyValue,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.12_stream_properties",
			Version: "4.0",
			Suite:   v4_0.StreamProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.13_type_casting",
			Version: "4.0",
			Suite:   v4_0.TypeCasting,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.14_url_encoding",
			Version: "4.0",
			Suite:   v4_0.URLEncoding,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.15_entity_references",
			Version: "4.0",
			Suite:   v4_0.EntityReferences,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.16_singleton_operations",
			Version: "4.0",
			Suite:   v4_0.SingletonOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.1_filter_string_functions",
			Version: "4.0",
			Suite:   v4_0.FilterStringFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.2_filter_date_functions",
			Version: "4.0",
			Suite:   v4_0.FilterDateFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.3_filter_arithmetic_functions",
			Version: "4.0",
			Suite:   v4_0.FilterArithmeticFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.4_filter_type_functions",
			Version: "4.0",
			Suite:   v4_0.FilterTypeFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.5_filter_logical_operators",
			Version: "4.0",
			Suite:   v4_0.FilterLogicalOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.6_filter_comparison_operators",
			Version: "4.0",
			Suite:   v4_0.FilterComparisonOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.7_filter_geo_functions",
			Version: "4.0",
			Suite:   v4_0.FilterGeoFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.8_filter_expanded_properties",
			Version: "4.0",
			Suite:   v4_0.FilterExpandedProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.9_string_function_edge_cases",
			Version: "4.0",
			Suite:   v4_0.StringFunctionEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.1_requesting_entities",
			Version: "4.0",
			Suite:   v4_0.RequestingEntities,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.2_create_entity",
			Version: "4.0",
			Suite:   v4_0.CreateEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.2.1_odata_bind",
			Version: "4.0",
			Suite:   v4_0.ODataBind,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.3_update_entity",
			Version: "4.0",
			Suite:   v4_0.UpdateEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.4_delete_entity",
			Version: "4.0",
			Suite:   v4_0.DeleteEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.5_upsert",
			Version: "4.0",
			Suite:   v4_0.Upsert,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.6_relationships",
			Version: "4.0",
			Suite:   v4_0.Relationships,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.6.1_navigation_property_operations",
			Version: "4.0",
			Suite:   v4_0.NavigationPropertyOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.7_deep_insert",
			Version: "4.0",
			Suite:   v4_0.DeepInsert,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.8_modify_relationships",
			Version: "4.0",
			Suite:   v4_0.ModifyRelationships,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.9_batch_requests",
			Version: "4.0",
			Suite:   v4_0.BatchRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.9.1_batch_error_handling",
			Version: "4.0",
			Suite:   v4_0.BatchErrorHandling,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.10_asynchronous_requests",
			Version: "4.0",
			Suite:   v4_0.AsynchronousRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.11_head_requests",
			Version: "4.0",
			Suite:   v4_0.HEADRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.12_returning_results",
			Version: "4.0",
			Suite:   v4_0.ReturningResults,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.13_action_function_parameters",
			Version: "4.0",
			Suite:   v4_0.ActionFunctionParameters,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.14_null_value_handling",
			Version: "4.0",
			Suite:   v4_0.NullValueHandling,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.15_data_validation",
			Version: "4.0",
			Suite:   v4_0.DataValidation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.5.1_conditional_requests",
			Version: "4.0",
			Suite:   v4_0.ConditionalRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.6_annotations",
			Version: "4.0",
			Suite:   v4_0.InstanceAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "12.1_operations",
			Version: "4.0",
			Suite:   v4_0.Operations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "13.1_asynchronous_processing",
			Version: "4.0",
			Suite:   v4_0.AsynchronousProcessing,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "14.1_vocabulary_annotations",
			Version: "4.0",
			Suite:   v4_0.VocabularyAnnotations,
		})
	}

	// TODO: Register v4.01 tests when implemented
	// if *version == "all" || *version == "4.01" {
	//     testSuites = append(testSuites, ...)
	// }

	if len(testSuites) == 0 {
		fmt.Println("No test suites found for version:", *version)
		os.Exit(1)
	}

	// Run tests
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()

	totalSuites := 0
	passedSuites := 0
	totalTests := 0
	passedTests := 0
	failedTests := 0
	skippedTests := 0

	for _, suiteInfo := range testSuites {
		// Skip if pattern is specified and doesn't match
		if *pattern != "" && !strings.Contains(suiteInfo.Name, *pattern) {
			continue
		}

		totalSuites++
		suite := suiteInfo.Suite()
		suite.ServerURL = *serverURL
		suite.Debug = *debug

		versionPrefix := "V4"
		if suiteInfo.Version == "4.01" {
			versionPrefix = "V4.01"
		}

		fmt.Printf("\033[0;34mRunning: [%s] %s\033[0m\n", versionPrefix, suiteInfo.Name)
		fmt.Println("─────────────────────────────────────────────────────────")

		err := suite.Run()

		totalTests += suite.Results.Total
		passedTests += suite.Results.Passed
		failedTests += suite.Results.Failed
		skippedTests += suite.Results.Skipped

		if err == nil {
			passedSuites++
		}

		fmt.Println()
	}

	// Print overall summary
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║                  OVERALL SUMMARY                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Test Scripts: %d/%d passed (%.0f%%)\n", passedSuites, totalSuites,
		float64(passedSuites)/float64(totalSuites)*100)
	fmt.Println("Individual Tests:")
	fmt.Printf("  - Total: %d\n", totalTests)
	fmt.Printf("  - Passing: %d\n", passedTests)
	fmt.Printf("  - Failing: %d\n", failedTests)
	fmt.Printf("  - Skipped: %d\n", skippedTests)
	if totalTests > 0 {
		fmt.Printf("  - Pass Rate: %.0f%%\n", float64(passedTests)/float64(totalTests)*100)
	}
	fmt.Println()

	if passedSuites == totalSuites {
		fmt.Println("\033[0;32m✓ ALL TESTS PASSED\033[0m")
		fmt.Println()
		os.Exit(0)
	} else {
		fmt.Println("\033[0;31m✗ SOME TESTS FAILED\033[0m")
		fmt.Println()
		os.Exit(1)
	}
}

func startComplianceServer() (*exec.Cmd, error) {
	fmt.Println("Starting compliance server...")

	// Find the project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	complianceServerPath := filepath.Join(projectRoot, "cmd", "complianceserver")

	// Build the compliance server
	fmt.Println("Building compliance server...")
	buildCmd := exec.Command("go", "build", "-o", "/tmp/complianceserver", ".")
	buildCmd.Dir = complianceServerPath
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build output:\n%s\n", string(buildOutput))
		return nil, fmt.Errorf("failed to build compliance server: %w", err)
	}

	// Determine database arguments
	dbArgs := []string{}
	if *dbType == "postgres" {
		dsn := *dbDSN
		if dsn == "" {
			dsn = os.Getenv("DATABASE_URL")
			if dsn == "" {
				dsn = "postgresql://odata:odata_dev@localhost:5432/odata_test?sslmode=disable"
			}
		}
		dbArgs = append(dbArgs, "-db", "postgres", "-dsn", dsn)
	} else {
		dsn := *dbDSN
		if dsn == "" {
			dsn = ":memory:"
		}
		dbArgs = append(dbArgs, "-db", "sqlite", "-dsn", dsn)
	}

	// Start the server
	fmt.Printf("Starting compliance server (db=%s)\n", *dbType)
	cmd := exec.Command("/tmp/complianceserver", dbArgs...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	fmt.Printf("Compliance server started (PID: %d)\n", cmd.Process.Pid)
	fmt.Println("Waiting for server to be ready...")

	// Wait for server to be ready (up to 60 seconds)
	for i := 0; i < 60; i++ {
		if checkServerConnectivity() {
			fmt.Println("\033[0;32m✓ Server is ready!\033[0m")
			fmt.Println()
			return cmd, nil
		}
		time.Sleep(1 * time.Second)
	}

	// Server failed to start, kill it
	//nolint:errcheck
	_ = cmd.Process.Kill()
	return nil, fmt.Errorf("server failed to start within 60 seconds")
}

func stopComplianceServer(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		fmt.Println()
		fmt.Printf("Stopping compliance server (PID: %d)...\n", cmd.Process.Pid)
		// Intentionally ignoring errors during cleanup
		//nolint:errcheck
		_ = cmd.Process.Kill()
		//nolint:errcheck
		_ = cmd.Wait()
		fmt.Println("Server stopped.")
	}
}

func checkServerConnectivity() bool {
	resp, err := framework.NewTestSuite("", "", "").Client.Get(*serverURL + "/")
	if err != nil {
		return false
	}
	//nolint:errcheck
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == 200
}

func findProjectRoot() (string, error) {
	// Start from current directory and walk up to find the go-odata project root
	// We need to skip the compliance-suite's own go.mod and find the parent project
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// First, move up to parent if we're in compliance-suite
	if filepath.Base(dir) == "compliance-suite" {
		dir = filepath.Dir(dir)
	}

	// Now look for cmd/complianceserver to verify this is the right directory
	complianceServerPath := filepath.Join(dir, "cmd", "complianceserver")
	if _, err := os.Stat(complianceServerPath); err == nil {
		return dir, nil
	}

	// If not found, walk up looking for go.mod with cmd/complianceserver
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			complianceServerPath := filepath.Join(dir, "cmd", "complianceserver")
			if _, err := os.Stat(complianceServerPath); err == nil {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root with cmd/complianceserver")
		}
		dir = parent
	}
}
