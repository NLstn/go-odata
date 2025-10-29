package handlers

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/response"
)

// WriteError writes an OData error response and logs if the write fails.
func WriteError(w http.ResponseWriter, status int, code, detail string) {
	if err := response.WriteError(w, status, code, detail); err != nil {
		fmt.Printf(LogMsgErrorWritingErrorResponse, err)
	}
}
