package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/nlstn/go-odata/internal/hookerrors"
	"github.com/nlstn/go-odata/internal/odataerrors"
)

func TestBuildODataErrorResponse_ODataError(t *testing.T) {
	status, errPayload := BuildODataErrorResponse(&odataerrors.ODataError{
		StatusCode: http.StatusBadRequest,
		Code:       odataerrors.ErrorCodeBadRequest,
		Message:    "validation failed",
		Target:     "Products(1)/Name",
		Details: []odataerrors.ErrorDetail{{
			Code:    "Required",
			Target:  "Name",
			Message: "Name is required",
		}},
	}, http.StatusInternalServerError, "Internal error")

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if errPayload.Code != "BadRequest" {
		t.Fatalf("code = %q, want %q", errPayload.Code, "BadRequest")
	}
	if errPayload.Target != "Products(1)/Name" {
		t.Fatalf("target = %q, want %q", errPayload.Target, "Products(1)/Name")
	}
	if len(errPayload.Details) != 1 || errPayload.Details[0].Code != "Required" {
		t.Fatalf("details = %#v, expected mapped detail", errPayload.Details)
	}
}

func TestBuildODataErrorResponse_HookErrorCustomCode(t *testing.T) {
	status, errPayload := BuildODataErrorResponse(&hookerrors.HookError{
		StatusCode: http.StatusConflict,
		Code:       "INVITE_ALREADY_MEMBER",
		Message:    "already a member",
		Target:     "members",
		Details: []hookerrors.ErrorDetail{{
			Code:    "MEMBERSHIP",
			Target:  "members",
			Message: "already present",
		}},
	}, http.StatusForbidden, "Authorization failed")

	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d", status, http.StatusConflict)
	}
	if errPayload.Code != "INVITE_ALREADY_MEMBER" {
		t.Fatalf("code = %q, want %q", errPayload.Code, "INVITE_ALREADY_MEMBER")
	}
	if errPayload.Target != "members" {
		t.Fatalf("target = %q, want %q", errPayload.Target, "members")
	}
	if len(errPayload.Details) != 1 || errPayload.Details[0].Code != "MEMBERSHIP" {
		t.Fatalf("details = %#v, expected hook details", errPayload.Details)
	}
}

func TestBuildODataErrorResponse_HookErrorFallback(t *testing.T) {
	status, errPayload := BuildODataErrorResponse(&hookerrors.HookError{
		Err: errors.New("wrapped detail"),
	}, http.StatusUnauthorized, "Unauthorized")

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
	}
	if errPayload.Code != "401" {
		t.Fatalf("code = %q, want %q", errPayload.Code, "401")
	}
	if errPayload.Message != "Unauthorized" {
		t.Fatalf("message = %q, want %q", errPayload.Message, "Unauthorized")
	}
	if len(errPayload.Details) != 1 || errPayload.Details[0].Message != "wrapped detail" {
		t.Fatalf("details = %#v, expected wrapped detail", errPayload.Details)
	}
}

func TestBuildODataErrorResponse_GenericFallback(t *testing.T) {
	status, errPayload := BuildODataErrorResponse(errors.New("boom"), http.StatusBadRequest, "Invalid query options")

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if errPayload.Code != "400" {
		t.Fatalf("code = %q, want %q", errPayload.Code, "400")
	}
	if errPayload.Message != "Invalid query options" {
		t.Fatalf("message = %q, want %q", errPayload.Message, "Invalid query options")
	}
	if len(errPayload.Details) != 1 || errPayload.Details[0].Message != "boom" {
		t.Fatalf("details = %#v, expected boom detail", errPayload.Details)
	}
}