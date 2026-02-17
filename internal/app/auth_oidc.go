package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type OIDCValidator struct {
	issuer   string
	audience string
	jwksURI  string
	cache    *jwk.Cache
}

func NewOIDCValidator(ctx context.Context, authority string, audience string) (*OIDCValidator, error) {
	authority = strings.TrimRight(strings.TrimSpace(authority), "/")
	if authority == "" {
		return nil, errors.New("authority is required")
	}

	audience = strings.TrimSpace(audience)
	if audience == "" {
		return nil, errors.New("audience is required")
	}

	jwksURI, err := discoverJWKSURI(ctx, authority)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	cache := jwk.NewCache(ctx)
	if err := cache.Register(
		jwksURI,
		jwk.WithMinRefreshInterval(5*time.Minute),
		jwk.WithHTTPClient(httpClient),
	); err != nil {
		return nil, fmt.Errorf("failed to register jwks cache: %w", err)
	}

	if _, err := cache.Refresh(ctx, jwksURI); err != nil {
		return nil, fmt.Errorf("failed to initialize jwks cache: %w", err)
	}

	validator := &OIDCValidator{
		issuer:   authority,
		audience: audience,
		jwksURI:  jwksURI,
		cache:    cache,
	}

	return validator, nil
}

func discoverJWKSURI(ctx context.Context, authority string) (string, error) {
	url := authority + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create discovery request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch oidc discovery document: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oidc discovery returned status %d", res.StatusCode)
	}

	var payload struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode oidc discovery document: %w", err)
	}

	payload.JWKSURI = strings.TrimSpace(payload.JWKSURI)
	if payload.JWKSURI == "" {
		return "", errors.New("oidc discovery document does not contain jwks_uri")
	}

	return payload.JWKSURI, nil
}

func (v *OIDCValidator) Validate(ctx context.Context, token string) (TokenClaims, error) {
	keySet, err := v.cache.Get(ctx, v.jwksURI)
	if err != nil {
		return TokenClaims{}, fmt.Errorf("failed to get jwks keyset: %w", err)
	}

	parsedToken, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	)
	if err != nil {
		if _, refreshErr := v.cache.Refresh(ctx, v.jwksURI); refreshErr == nil {
			if refreshedSet, getErr := v.cache.Get(ctx, v.jwksURI); getErr == nil {
				parsedToken, err = jwt.Parse(
					[]byte(token),
					jwt.WithKeySet(refreshedSet),
					jwt.WithValidate(true),
				)
			}
		}
	}
	if err != nil {
		return TokenClaims{}, fmt.Errorf("token verification failed: %w", err)
	}

	if err := jwt.Validate(
		parsedToken,
		jwt.WithAudience(v.audience),
	); err != nil {
		return TokenClaims{}, fmt.Errorf("token claims validation failed: %w", err)
	}

	if normalizeIssuer(parsedToken.Issuer()) != normalizeIssuer(v.issuer) {
		return TokenClaims{}, errors.New("token claims validation failed: issuer mismatch")
	}

	scopes := extractSupportedScopes(parsedToken)
	return TokenClaims{Scopes: scopes}, nil
}

func extractSupportedScopes(token jwt.Token) []string {
	result := make([]string, 0)

	if rawScope, ok := token.Get("scope"); ok {
		if scope, cast := rawScope.(string); cast {
			for _, item := range strings.Fields(scope) {
				result = append(result, item)
			}
		}
	}

	if rawPermissions, ok := token.Get("permissions"); ok {
		switch permissions := rawPermissions.(type) {
		case []string:
			result = append(result, permissions...)
		case []any:
			for _, item := range permissions {
				permission, cast := item.(string)
				if cast {
					result = append(result, strings.TrimSpace(permission))
				}
			}
		}
	}

	return unique(result)
}

func unique(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizeIssuer(issuer string) string {
	return strings.TrimRight(strings.TrimSpace(issuer), "/")
}
