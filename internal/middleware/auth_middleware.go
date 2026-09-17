package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/domain/rbac"
	"github.com/ramadhanrzq/backend-go/internal/domain/user"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

type contextKey string

const (
	claimsContextKey      contextKey = "claims"
	permissionsContextKey contextKey = "permissions"
)

// RequireAuth memvalidasi header "Authorization: Bearer <token>" sebelum
// request diteruskan. Claims user yang valid tersuntik ke request context.
func RequireAuth(auth user.AuthService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := authenticate(auth, r)
		if err != nil {
			if strings.Contains(err.Error(), "header") {
				w.Header().Set("WWW-Authenticate", `Bearer realm="lattepos"`)
			} else {
				w.Header().Set("WWW-Authenticate", `Bearer realm="lattepos", error="invalid_token"`)
			}
			response.Error(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// RequirePermission memvalidasi JWT dan memastikan user memiliki permission yang dibutuhkan.
// Permission set di-cache dalam request context untuk menghindari DB query berulang dalam request yang sama.
func RequirePermission(auth user.AuthService, rbacSvc rbac.RBACService, permissionName string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			var err error
			claims, err = authenticate(auth, r)
			if err != nil {
				if strings.Contains(err.Error(), "header") {
					w.Header().Set("WWW-Authenticate", `Bearer realm="lattepos"`)
				} else {
					w.Header().Set("WWW-Authenticate", `Bearer realm="lattepos", error="invalid_token"`)
				}
				response.Error(w, http.StatusUnauthorized, err.Error())
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), claimsContextKey, claims))
		}

		permSet, ok := PermissionsFromContext(r.Context())
		if !ok {
			var err error
			permSet, err = rbacSvc.GetUserPermissions(r.Context(), claims.UserID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "Gagal memverifikasi izin akses")
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), permissionsContextKey, permSet))
		}

		if _, allowed := permSet[permissionName]; !allowed {
			response.Error(w, http.StatusForbidden, "Forbidden: permission required: "+permissionName)
			return
		}

		next(w, r.WithContext(r.Context()))
	}
}

func authenticate(auth user.AuthService, r *http.Request) (*appjwt.Claims, error) {
	const prefix = "Bearer "

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, prefix) {
		return nil, http.ErrNoCookie // standard sentinel placeholder for missing token
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	return auth.VerifyToken(tokenString)
}

// ClaimsFromContext mengambil claims user yang disuntik oleh RequireAuth.
func ClaimsFromContext(ctx context.Context) (*appjwt.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*appjwt.Claims)
	return claims, ok
}

// PermissionsFromContext mengambil cache permission set user dari request context.
func PermissionsFromContext(ctx context.Context) (map[string]struct{}, bool) {
	perms, ok := ctx.Value(permissionsContextKey).(map[string]struct{})
	return perms, ok
}
