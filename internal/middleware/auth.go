package middleware

import (
	"context"
	"net/http"
	"strings"

	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// TokenVerifier adalah port yang dibutuhkan middleware untuk memvalidasi JWT.
// Dipenuhi oleh service module auth.
type TokenVerifier interface {
	VerifyToken(tokenString string) (*appjwt.Claims, error)
}

// PermissionChecker adalah port yang dibutuhkan middleware untuk memeriksa izin user.
// orgID kosong berarti pengecekan di scope global; dipenuhi oleh service module rbac.
type PermissionChecker interface {
	GetUserPermissions(ctx context.Context, orgID, userID string) (map[string]struct{}, error)
}

type contextKey string

const (
	claimsContextKey      contextKey = "claims"
	permissionsContextKey contextKey = "permissions"
)

// RequireAuth memvalidasi header "Authorization: Bearer <token>" sebelum
// request diteruskan. Claims user yang valid tersuntik ke request context.
func RequireAuth(auth TokenVerifier, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := authenticate(auth, r)
		if err != nil {
			writeUnauthorized(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// RequirePermission memvalidasi JWT dan memastikan user memiliki permission yang dibutuhkan
// pada scope request: organisasi di path bila ada, atau global bila tidak ada.
// Permission set di-cache dalam request context untuk menghindari DB query berulang.
func RequirePermission(auth TokenVerifier, rbacSvc PermissionChecker, permissionName string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			var err error
			claims, err = authenticate(auth, r)
			if err != nil {
				writeUnauthorized(w, err)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), claimsContextKey, claims))
		}

		orgID, _ := OrgIDFromContext(r.Context())

		permSet, ok := PermissionsFromContext(r.Context())
		if !ok {
			var err error
			permSet, err = rbacSvc.GetUserPermissions(r.Context(), orgID, claims.UserID)
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

func writeUnauthorized(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), "header") {
		w.Header().Set("WWW-Authenticate", `Bearer realm="lattepos"`)
	} else {
		w.Header().Set("WWW-Authenticate", `Bearer realm="lattepos", error="invalid_token"`)
	}
	response.Error(w, http.StatusUnauthorized, err.Error())
}

func authenticate(auth TokenVerifier, r *http.Request) (*appjwt.Claims, error) {
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
