package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

type claimsContextKey struct{}

// WithClaims returns a derived context carrying the authenticated JWT claims.
func WithClaims(ctx context.Context, claims *domain.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext extracts the authenticated JWT claims from ctx, if present.
func ClaimsFromContext(ctx context.Context) (*domain.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(*domain.Claims)
	return claims, ok && claims != nil
}

// RequireAuth returns an HTTP middleware that validates the Bearer JWT token via JwtService.
func RequireAuth(jwtSvc *infrastructure.JwtService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RequireAuthFunc(jwtSvc, next.ServeHTTP)
	}
}

// RequireAuthFunc wraps an http.HandlerFunc with Bearer JWT validation.
func RequireAuthFunc(jwtSvc *infrastructure.JwtService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawToken := extractBearer(r.Header.Get("Authorization"))
		if rawToken == "" || jwtSvc == nil {
			httpx.Unauthorized(w, "missing or invalid Authorization Bearer token")
			return
		}

		claims, err := jwtSvc.ValidateAccessToken(rawToken)
		if err != nil {
			httpx.Unauthorized(w, "invalid or expired access token")
			return
		}

		next(w, r.WithContext(WithClaims(r.Context(), claims)))
	}
}

// RequireSuperAdmin returns an HTTP middleware that requires an authenticated SUPER_ADMIN JWT.
// Non-SUPER_ADMIN users receive 403 Forbidden.
func RequireSuperAdmin(jwtSvc *infrastructure.JwtService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RequireSuperAdminFunc(jwtSvc, next.ServeHTTP)
	}
}

// RequireSuperAdminFunc wraps an http.HandlerFunc, enforcing both valid JWT authentication and SUPER_ADMIN role.
func RequireSuperAdminFunc(jwtSvc *infrastructure.JwtService, next http.HandlerFunc) http.HandlerFunc {
	return RequireAuthFunc(jwtSvc, func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			httpx.Unauthorized(w, "authentication required")
			return
		}
		if claims.Role != domain.RoleSuperAdmin {
			httpx.Forbidden(w, domain.ErrForbiddenSuperAdminOnly.Error())
			return
		}
		next(w, r)
	})
}

func extractBearer(headerVal string) string {
	trimmed := strings.TrimSpace(headerVal)
	if len(trimmed) < 7 || !strings.EqualFold(trimmed[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(trimmed[7:])
}
