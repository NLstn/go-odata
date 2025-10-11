package handlers

// HTTP header constants
const (
	HeaderContentType       = "Content-Type"
	HeaderODataVersion      = "OData-Version"
	HeaderPreferenceApplied = "Preference-Applied"
	HeaderIfMatch           = "If-Match"
	HeaderETag              = "ETag"
)

// Content type constants
const (
	ContentTypeJSON      = "application/json;odata.metadata=minimal"
	ContentTypePlainText = "text/plain; charset=utf-8"
)

// Error message constants
const (
	ErrMsgMethodNotAllowed      = "Method not allowed"
	ErrMsgInvalidQueryOptions   = "Invalid query options"
	ErrMsgDatabaseError         = "Database error"
	ErrMsgInvalidRequestBody    = "Invalid request body"
	ErrMsgInvalidKey            = "Invalid key"
	ErrMsgEntityNotFound        = "Entity not found"
	ErrMsgInternalError         = "Internal error"
	ErrMsgPreconditionFailed    = "Precondition failed"
	ErrDetailPreconditionFailed = "The entity has been modified. Please refresh and try again."
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
