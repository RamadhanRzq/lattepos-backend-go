package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/domain/organization"
	"github.com/ramadhanrzq/backend-go/internal/domain/user"
	"github.com/ramadhanrzq/backend-go/pkg/response"
)

const orgContextKey contextKey = "org"

// RequireOrgMember memvalidasi JWT dan keanggotaan user dalam organisasi berdasarkan path slug atau JWT claims.
func RequireOrgMember(authSvc user.AuthService, orgSvc organization.OrgService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			var err error
			claims, err = authenticate(authSvc, r)
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

		slug := strings.TrimSpace(r.PathValue("slug"))
		if slug == "" {
			slug = claims.OrgSlug
		}

		if slug == "" {
			response.Error(w, http.StatusBadRequest, "Organization slug required")
			return
		}

		org, err := orgSvc.GetBySlug(r.Context(), slug)
		if err != nil {
			response.Error(w, http.StatusNotFound, "Organization not found")
			return
		}

		isMember, err := orgSvc.IsUserMember(r.Context(), org.ID, claims.UserID)
		if err != nil || !isMember {
			response.Error(w, http.StatusForbidden, "Forbidden: you are not a member of this organization")
			return
		}

		ctx := context.WithValue(r.Context(), orgContextKey, org)
		next(w, r.WithContext(ctx))
	}
}

// OrgFromContext mengambil entity organization yang tersuntik oleh RequireOrgMember.
func OrgFromContext(ctx context.Context) (*organization.Organization, bool) {
	org, ok := ctx.Value(orgContextKey).(*organization.Organization)
	return org, ok
}
