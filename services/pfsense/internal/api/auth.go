package api

import (
	"context"
	"net/http"
	"os"
	"slices"
	"strings"
)

type contextKey string

const principalContextKey contextKey = "principal"

type AuthMiddleware struct {
	mode          string
	requiredGroup string
}

func NewAuthMiddlewareFromEnv() AuthMiddleware {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_MODE")))
	if mode == "" {
		mode = "enforced"
	}

	requiredGroup := strings.TrimSpace(os.Getenv("AUTH_REQUIRED_GROUP"))

	return AuthMiddleware{
		mode:          mode,
		requiredGroup: requiredGroup,
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

		if a.requiredGroup != "" {
			groups := a.readGroups(r)
			if !slices.Contains(groups, a.requiredGroup) {
				http.Error(w, "missing required group", http.StatusForbidden)
				return
			}
		}

		ctx := context.WithValue(r.Context(), principalContextKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a AuthMiddleware) readPrincipal(r *http.Request) string {
	for _, header := range []string{"X-Asterism-Principal", "X-Forwarded-User", "X-Forwarded-Email"} {
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
