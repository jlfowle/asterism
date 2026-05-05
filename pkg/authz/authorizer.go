package authz

import (
    "context"
    "os"
    "strings"
)

// Authorizer decides whether a principal is allowed to perform an action.
type Authorizer interface {
    // IsAllowed returns allowed (bool), a human reason string when denied, and an error for internal failures.
    IsAllowed(ctx context.Context, principal string, groups []string, resource string, verb string) (bool, string, error)
}

// NewAuthorizer chooses an implementation based on environment configuration.
// If AUTHZ_OPA_URL is set, an OPA-backed authorizer will be used. Otherwise a
// simple local group-based authorizer is returned.
func NewAuthorizer(requiredGroup string) Authorizer {
    opa := strings.TrimSpace(os.Getenv("AUTHZ_OPA_URL"))
    if opa != "" {
        return NewOPAAuthorizer(opa, requiredGroup)
    }

    return &LocalAuthorizer{requiredGroup: strings.TrimSpace(requiredGroup)}
}
