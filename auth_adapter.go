package odata

import "github.com/nlstn/go-odata/internal/auth"

// AuthContext contains authentication and request metadata for authorization decisions.
type AuthContext = auth.AuthContext

// RequestMetadata captures HTTP request details relevant to authorization.
type RequestMetadata = auth.RequestMetadata

// ResourceDescriptor describes the resource being accessed.
type ResourceDescriptor = auth.ResourceDescriptor

// Operation defines the type of action being authorized.
type Operation = auth.Operation

// Operation values for authorization checks.
const (
	OperationRead     = auth.OperationRead
	OperationCreate   = auth.OperationCreate
	OperationUpdate   = auth.OperationUpdate
	OperationDelete   = auth.OperationDelete
	OperationQuery    = auth.OperationQuery
	OperationMetadata = auth.OperationMetadata
)

// Decision represents the result of an authorization check.
type Decision = auth.Decision

// Policy defines the interface for authorization decisions.
type Policy = auth.Policy

// Allow returns an allow decision.
func Allow() Decision {
	return auth.Allow()
}

// Deny returns a deny decision with an optional reason.
func Deny(reason string) Decision {
	return auth.Deny(reason)
}

// SetPolicy registers an authorization policy for the service.
// Pass nil to clear the policy (all requests will be allowed).
//
// # Example
//
//	err := service.SetPolicy(myAuthPolicy)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (s *Service) SetPolicy(policy Policy) error {
	s.policy = policy
	if s.metadataHandler != nil {
		s.metadataHandler.SetPolicy(policy)
	}
	if s.serviceDocumentHandler != nil {
		s.serviceDocumentHandler.SetPolicy(policy)
	}
	if s.handlers != nil {
		for _, handler := range s.handlers {
			if handler != nil {
				handler.SetPolicy(policy)
			}
		}
	}
	return nil
}
