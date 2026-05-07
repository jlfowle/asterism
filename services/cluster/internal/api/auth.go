package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/jlfowle/asterism/pkg/authz"
)

type contextKey string

const principalContextKey contextKey = "principal"

type AuthMiddleware struct {
	mode          string
	requiredGroup string
	opaEnabled    bool
	authorizer    authz.Authorizer
}

func NewAuthMiddlewareFromEnv() AuthMiddleware {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_MODE")))
	if mode == "" {
		mode = "enforced"
	}

	requiredGroup := strings.TrimSpace(os.Getenv("AUTH_REQUIRED_GROUP"))
	opaURL := strings.TrimSpace(os.Getenv("AUTHZ_OPA_URL"))

	return AuthMiddleware{
		mode:          mode,
		requiredGroup: requiredGroup,
		opaEnabled:    opaURL != "",
		authorizer:    authz.NewAuthorizer(requiredGroup),
	}
}

func (a AuthMiddleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.mode == "disabled" {
			next.ServeHTTP(w, r)
			return
		}

		principal := a.readPrincipal(r)
		if principal == "" && a.mode == "enforced" {
			http.Error(w, "missing principal", http.StatusUnauthorized)
			return
		}

		if a.requiredGroup != "" || a.opaEnabled {
			groups := a.readGroups(r)
			allowed, reason, err := a.authorizer.IsAllowed(r.Context(), principal, groups, r.URL.Path, r.Method)
			if err != nil {
				http.Error(w, fmt.Sprintf("authorization error: %v", err), http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, reason, http.StatusForbidden)
				return
			}
		}

		ctx := context.WithValue(r.Context(), principalContextKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a AuthMiddleware) readPrincipal(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-User", "X-Forwarded-Email"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value != "" {
			return value
		}
	}

	return ""
}

func (a AuthMiddleware) readGroups(r *http.Request) []string {
	raw := strings.TrimSpace(r.Header.Get("X-Asterism-Groups"))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	groups := make([]string, 0, len(parts))
	for _, group := range parts {
		trimmed := strings.TrimSpace(group)
		if trimmed != "" {
			groups = append(groups, trimmed)
		}
	}

	return groups
}

func principalFromContext(ctx context.Context) string {
	principal, ok := ctx.Value(principalContextKey).(string)
	if !ok {
		return ""
	}

	return principal
}
