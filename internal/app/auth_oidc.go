package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var ErrForbidden = errors.New("forbidden")

type OIDCValidator struct {
	issuer   string
	audience string
	scope    string
	keySet   jwk.Set
}

func NewOIDCValidator(ctx context.Context, issuer string, audience string, scope string, obs Observability) (*OIDCValidator, error) {
	if issuer == "" {
		return nil, errors.New("authority is required")
	}

	if audience == "" {
		return nil, errors.New("audience is required")
	}

	if scope == "" {
		return nil, errors.New("scope is required")
	}

	normalizedIssuer := strings.TrimRight(issuer, "/")
	jwksURI := normalizedIssuer + "/.well-known/jwks.json"

	keySet, err := jwk.Fetch(ctx, jwksURI, jwk.WithHTTPClient(obs.HttpClient(nil)))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jwks: %w", err)
	}
	if keySet.Len() == 0 {
		return nil, errors.New("jwks is empty")
	}

	validator := &OIDCValidator{
		issuer:   normalizedIssuer,
		audience: audience,
		scope:    scope,
		keySet:   keySet,
	}

	return validator, nil
}

func (v *OIDCValidator) Validate(ctx context.Context, token string) error {
	parsedToken, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(v.keySet),
		jwt.WithValidate(true),
	)
	if err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}

	if err := jwt.Validate(parsedToken, jwt.WithAudience(v.audience)); err != nil {
		return fmt.Errorf("token claims validation failed: %w", err)
	}

	if normalizeIssuer(parsedToken.Issuer()) != v.issuer {
		return errors.New("token claims validation failed: issuer mismatch")
	}

	if hasRequiredScope(parsedToken, v.scope) {
		return nil
	}

	return fmt.Errorf("%w: required scope %q missing", ErrForbidden, v.scope)
}

func hasRequiredScope(token jwt.Token, required string) bool {
	rawScope, ok := token.Get("scope")
	if !ok {
		return false
	}

	scope, cast := rawScope.(string)
	if !cast {
		return false
	}

	return slices.Contains(strings.Fields(scope), required)
}

func normalizeIssuer(issuer string) string {
	return strings.TrimRight(issuer, "/")
}
