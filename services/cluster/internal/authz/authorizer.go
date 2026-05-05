package authz

import (
    "context"
    "strings"
)

// Authorizer decides whether a principal is allowed to perform an action.
type Authorizer interface {
    // IsAllowed returns allowed (bool), a human reason string when denied, and an error for internal failures.
    IsAllowed(ctx context.Context, principal string, groups []string, resource string, verb string) (bool, string, error)
}

// NewAuthorizer returns a simple local authorizer. In future this factory
// may return an OPA-backed or remote authorizer based on environment config.
func NewAuthorizer(requiredGroup string) Authorizer {
    return &LocalAuthorizer{requiredGroup: strings.TrimSpace(requiredGroup)}
}
