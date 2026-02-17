package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

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

func NewOIDCValidator(ctx context.Context, authority string, audience string, scope string) (*OIDCValidator, error) {
	if authority == "" {
		return nil, errors.New("authority is required")
	}

	if audience == "" {
		return nil, errors.New("audience is required")
	}

	if scope == "" {
		return nil, errors.New("scope is required")

	}

	jwksURI := strings.TrimRight(authority, "/") + "/.well-known/jwks.json"

	keySet, err := jwk.Fetch(ctx, jwksURI, jwk.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jwks: %w", err)
	}
	if keySet.Len() == 0 {
		return nil, errors.New("jwks is empty")
	}

	validator := &OIDCValidator{
		issuer:   authority,
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

	if parsedToken.Issuer() != v.issuer {
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
