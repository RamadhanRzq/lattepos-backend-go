package middleware

import (
	"context"

	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

// NewContextWithOrgID returns a context carrying the org ID, usable by
// other packages' tests that need to simulate RequireOrgMember injection.
func NewContextWithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, orgIDContextKey, orgID)
}

// NewContextWithClaims returns a context carrying JWT claims, usable by
// other packages' tests that need to simulate RequireAuth injection.
func NewContextWithClaims(ctx context.Context, claims *appjwt.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}
