package query

import "errors"

// Pre-defined errors for common cases to reduce fmt.Errorf allocations
// These errors are used throughout the query package for better performance

var (
	// General errors
	errUnsupportedASTNodeType = errors.New("unsupported AST node type")
	errEntityMetadataIsNil    = errors.New("entity metadata is nil")
	errEntityMetadataRequired = errors.New("entity metadata is required")
	errEntityHasNoKeyProps    = errors.New("entity has no key properties")
	errInvalidSQLIdentifier   = errors.New("invalid SQL identifier in table or column name")
	errTableNameRequired      = errors.New("table name is required")

	// $apply errors
	errEmptyApplyString            = errors.New("empty apply string")
	errNoValidTransformations      = errors.New("no valid transformations found")
	errInvalidAggregateFormat      = errors.New("invalid aggregate format")
	errInvalidComputeFormat        = errors.New("invalid compute format")
	errInvalidFilterFormat         = errors.New("invalid filter format")
	errInvalidGroupByFormat        = errors.New("invalid groupby format")
	errNoValidAggregateExpressions = errors.New("no valid aggregate expressions found")
	errNoValidComputeExpressions   = errors.New("no valid compute expressions found")
	errInvalidResults              = errors.New("invalid results")
	errNilResults                  = errors.New("nil results")

	// Aggregate/Compute expression errors
	errInvalidAggregateExprFormat     = errors.New("invalid aggregate expression format, expected 'property with method as alias'")
	errInvalidAggregateExprMissingAs  = errors.New("invalid aggregate expression format, missing 'as alias'")
	errInvalidComputeExprFormat       = errors.New("invalid compute expression format, expected 'expression as alias'")
	errInvalidCountFormat             = errors.New("invalid $count format, expected '$count as alias'")
	errEmptyComputeExpression         = errors.New("empty compute expression")
	errGroupByPropsNeedParens         = errors.New("groupby properties must be in parentheses")
	errExpectedCommaAfterGroupByProps = errors.New("expected comma after groupby properties")

	// Parenthesis errors
	errMissingClosingParenAggregate    = errors.New("missing closing parenthesis in aggregate")
	errMissingClosingParenCompute      = errors.New("missing closing parenthesis in compute")
	errMissingClosingParenFilter       = errors.New("missing closing parenthesis in filter")
	errMissingClosingParenGroupBy      = errors.New("missing closing parenthesis in groupby")
	errMissingClosingParenGroupByProps = errors.New("missing closing parenthesis for groupby properties")

	// $expand errors
	errInvalidExpandSyntaxMissingQuote    = errors.New("invalid $expand syntax: missing closing quote")
	errInvalidExpandSyntaxMissingParen    = errors.New("invalid $expand syntax: missing ')'")
	errInvalidExpandSyntaxUnexpectedParen = errors.New("invalid $expand syntax: unexpected ')'")

	// $levels errors
	errLevelsMustBeIntOrMax       = errors.New("$levels must be a positive integer or 'max'")
	errLevelsMaxRequiresDepth     = errors.New("$levels=max requires a positive maximum expand depth")
	errNestedLevelsMustBeIntOrMax = errors.New("invalid nested $levels: must be a positive integer or 'max'")

	// Navigation metadata errors
	errNavMetadataMissingForCompute = errors.New("navigation target metadata is missing for $compute")
	errNavMetadataMissingForExpand  = errors.New("navigation target metadata is missing for $expand")
	errNavMetadataMissingForFilter  = errors.New("navigation target metadata is missing for $filter")
	errNavMetadataMissingForOrderBy = errors.New("navigation target metadata is missing for $orderby")
	errNavMetadataMissingForSelect  = errors.New("navigation target metadata is missing for $select")

	// Query option errors
	errSkipTokenAndSkipTogether = errors.New("$skiptoken and $skip cannot be used together")
	errInvalidCount             = errors.New("invalid $count: must be 'true' or 'false'")
	errNestedCountInvalid       = errors.New("invalid nested $count: must be 'true' or 'false'")
	errInvalidIndex             = errors.New("invalid $index: must not have a value")
	errInvalidSchemaVersion     = errors.New("invalid $schemaversion: schema version cannot be empty")
	errInvalidSearch            = errors.New("invalid $search: search query cannot be empty")

	// Parameter alias errors
	errEmptyAliasName = errors.New("invalid parameter alias: empty alias name")

	// Function argument errors
	errCastRequires2Args          = errors.New("function cast requires 2 arguments")
	errConcatRequires2Args        = errors.New("function concat requires 2 arguments")
	errSubstringRequires2Or3Args  = errors.New("function substring requires 2 or 3 arguments")
	errGeoDistanceRequires2Args   = errors.New("function geo.distance requires 2 arguments")
	errGeoIntersectsRequires2Args = errors.New("function geo.intersects requires 2 arguments")
	errGeoLengthRequires1Arg      = errors.New("function geo.length requires 1 argument")
	errIsOfRequires1Or2Args       = errors.New("function isof requires 1 or 2 arguments")

	// Function argument type errors
	errSubstringArgsMustBeLiterals          = errors.New("substring arguments must be literals")
	errSubstringStartNonNegative            = errors.New("substring start parameter must be non-negative")
	errSubstringLengthNonNegative           = errors.New("substring length parameter must be non-negative")
	errSecondArgOfCastMustBeType            = errors.New("second argument of cast must be a type name")
	errSecondArgOfIsOfMustBeType            = errors.New("second argument of isof must be a type name")
	errArgOfIsOfMustBeType                  = errors.New("argument of isof must be a type name")
	errFirstArgOfConcatMustBeLitPropFunc    = errors.New("first argument of concat must be a literal, property, or function")
	errSecondArgOfConcatMustBeLitPropFunc   = errors.New("second argument of concat must be a literal, property, or function")
	errSecondArgOfGeoDistanceMustBeGeoLit   = errors.New("second argument of geo.distance must be a geography or geometry literal")
	errSecondArgOfGeoIntersectsMustBeGeoLit = errors.New("second argument of geo.intersects must be a geography or geometry literal")

	// Comparison/arithmetic errors
	errLeftSideOfCompMustBeProp           = errors.New("left side of comparison must be a property name or arithmetic expression")
	errRightSideOfCompMustBeLitPropFunc   = errors.New("right side of comparison must be a literal, property, or function")
	errRightSideOfArithMustBeLitPropArith = errors.New("right side of arithmetic expression must be a literal, property, or arithmetic expression")

	// Lambda errors
	errLambdaCollMustBePropPath = errors.New("lambda collection must be a property path")

	// Collection errors
	errCollectionValuesMustBeLiterals = errors.New("collection values must be literals")
	errInOperatorRequiresCollection   = errors.New("'in' operator requires a collection on the right side")

	// FTS errors
	errFTSNotAvailable = errors.New("FTS is not available")

	// Literal errors
	errNumericLiteralOutOfRange = errors.New("numeric literal value out of range for Edm.Int64")

	// $compute transformation errors
	errInvalidComputeFailedToParse = errors.New("invalid $compute: failed to parse compute transformation")

	// Navigation path errors
	errNavPathNoRemainingProperty = errors.New("navigation path has no remaining property")
	errNavPathEndsWithNavProp     = errors.New("navigation path ends with a navigation property")

	// Orderby errors
	errOrderByInvalidDirection = errors.New("invalid direction for property, expected 'asc' or 'desc'")

	// Function argument count errors
	errFunctionRequires0Args = errors.New("function requires 0 arguments")
	errFunctionRequires1Arg  = errors.New("function requires 1 argument")
	errFunctionRequires2Args = errors.New("function requires 2 arguments")

	// Unary operator errors
	errUnsupportedUnaryOperator = errors.New("unsupported unary operator")

	// Lambda parsing errors
	errExpectedColonAfterLambdaVar = errors.New("expected ':' after lambda range variable")
	errFailedToParseLambdaPred     = errors.New("failed to parse lambda predicate")
	errFailedToConvertLambdaPred   = errors.New("failed to convert lambda predicate")

	// Expected token errors
	errExpectedLParen        = errors.New("expected '(' after 'in' operator")
	errExpectedIdentAfterNav = errors.New("expected identifier after '/' in property path")

	// Second argument type errors
	errSecondArgMustBeLitOrProp      = errors.New("second argument must be a literal or property")
	errSecondArgMustBeLitPropOrFunc  = errors.New("second argument must be a literal, property, or function")
	errFirstArgMustBePropOrFuncCall  = errors.New("first argument must be a property name or function call")
	errUnsupportedGeospatialFunction = errors.New("unsupported geospatial function")
)
