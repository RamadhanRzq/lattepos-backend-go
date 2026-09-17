package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ramadhanrzq/backend-go/pkg/response"
)

// OrgMembership adalah port yang dibutuhkan middleware untuk resolusi organisasi
// dan pemeriksaan keanggotaan. Dipenuhi oleh service module organizations.
type OrgMembership interface {
	ResolveOrgID(ctx context.Context, slug string) (string, error)
	IsUserMember(ctx context.Context, orgID, userID string) (bool, error)
}

const (
	orgIDContextKey   contextKey = "org_id"
	orgSlugContextKey contextKey = "org_slug"
)

// RequireOrgMember memvalidasi JWT dan keanggotaan user dalam organisasi
// berdasarkan path slug atau JWT claims.
func RequireOrgMember(auth TokenVerifier, orgs OrgMembership, next http.HandlerFunc) http.HandlerFunc {
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

		slug := strings.TrimSpace(r.PathValue("slug"))
		if slug == "" {
			slug = claims.OrgSlug
		}

		if slug == "" {
			response.Error(w, http.StatusBadRequest, "Organization slug required")
			return
		}

		orgID, err := orgs.ResolveOrgID(r.Context(), slug)
		if err != nil {
			response.Error(w, http.StatusNotFound, "Organization not found")
			return
		}

		isMember, err := orgs.IsUserMember(r.Context(), orgID, claims.UserID)
		if err != nil || !isMember {
			response.Error(w, http.StatusForbidden, "Forbidden: you are not a member of this organization")
			return
		}

		ctx := context.WithValue(r.Context(), orgIDContextKey, orgID)
		ctx = context.WithValue(ctx, orgSlugContextKey, slug)
		next(w, r.WithContext(ctx))
	}
}

// OrgIDFromContext mengambil ID organisasi yang tersuntik oleh RequireOrgMember.
func OrgIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(orgIDContextKey).(string)
	return id, ok
}
