package authz

import (
    "context"
)

// LocalAuthorizer implements a minimal group-based check.
type LocalAuthorizer struct{
    requiredGroup string
}

// IsAllowed returns true when no required group is configured or when the
// principal's groups contain the required group.
func (l *LocalAuthorizer) IsAllowed(ctx context.Context, principal string, groups []string, resource string, verb string) (bool, string, error) {
    if l.requiredGroup == "" {
        return true, "", nil
    }

    for _, g := range groups {
        if g == l.requiredGroup {
            return true, "", nil
        }
    }

    return false, "missing required group", nil
}
