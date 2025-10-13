package handlers

// HTTP header constants
const (
	HeaderContentType       = "Content-Type"
	HeaderODataVersion      = "OData-Version"
	HeaderODataMaxVersion   = "OData-MaxVersion"
	HeaderPreferenceApplied = "Preference-Applied"
	HeaderIfMatch           = "If-Match"
	HeaderIfNoneMatch       = "If-None-Match"
	HeaderETag              = "ETag"
)

// Content type constants
const (
	ContentTypeJSON      = "application/json;odata.metadata=minimal"
	ContentTypePlainText = "text/plain; charset=utf-8"
)

// Error message constants
const (
	ErrMsgMethodNotAllowed       = "Method not allowed"
	ErrMsgInvalidQueryOptions    = "Invalid query options"
	ErrMsgDatabaseError          = "Database error"
	ErrMsgInvalidRequestBody     = "Invalid request body"
	ErrMsgInvalidKey             = "Invalid key"
	ErrMsgEntityNotFound         = "Entity not found"
	ErrMsgInternalError          = "Internal error"
	ErrMsgPreconditionFailed     = "Precondition failed"
	ErrDetailPreconditionFailed  = "The entity has been modified. Please refresh and try again."
	ErrMsgVersionNotSupported    = "OData version not supported"
	ErrDetailVersionNotSupported = "This service only supports OData version 4.0 and above. The maximum version specified in the OData-MaxVersion header is below 4.0."
	ErrMsgValidationFailed       = "Validation failed"
)

// Error detail format constants
const (
	ErrDetailFailedToParseJSON = "Failed to parse JSON: %v"
)

// Log message format constants
const (
	LogMsgErrorWritingErrorResponse  = "Error writing error response: %v\n"
	LogMsgErrorWritingEntityResponse = "Error writing entity response: %v\n"
)

// OData format constants
const (
	ODataContextFormat   = "%s/$metadata#%s/$entity"
	ODataEntityKeyFormat = "%s(%s)"
	ODataContextProperty = "@odata.context"
	EntityKeyNotExistFmt = "The entity with key '%s' does not exist"
)
