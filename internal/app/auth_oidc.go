package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type OIDCValidator struct {
	issuer     string
	audiences  map[string]struct{}
	jwksURI    string
	httpClient *http.Client

	mu              sync.RWMutex
	keySet          jwk.Set
	lastFetch       time.Time
	refreshInterval time.Duration
}

func NewOIDCValidator(ctx context.Context, authority string, issuer string, audiences []string) (*OIDCValidator, error) {
	authority = strings.TrimRight(strings.TrimSpace(authority), "/")
	if authority == "" {
		return nil, errors.New("authority is required")
	}

	audienceMap := make(map[string]struct{}, len(audiences))
	for _, aud := range audiences {
		normalized := strings.TrimSpace(aud)
		if normalized == "" {
			continue
		}
		audienceMap[normalized] = struct{}{}
	}
	if len(audienceMap) == 0 {
		return nil, errors.New("at least one audience is required")
	}

	normalizedIssuer := strings.TrimSpace(strings.TrimRight(issuer, "/"))
	if normalizedIssuer == "" {
		normalizedIssuer = authority
	}

	jwksURI, err := discoverJWKSURI(ctx, authority)
	if err != nil {
		return nil, err
	}

	validator := &OIDCValidator{
		issuer:          normalizedIssuer,
		audiences:       audienceMap,
		jwksURI:         jwksURI,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		refreshInterval: 5 * time.Minute,
	}

	if err := validator.refreshKeySet(ctx); err != nil {
		return nil, err
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
	keySet, err := v.getKeySet(ctx)
	if err != nil {
		return TokenClaims{}, err
	}

	parsedToken, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	)
	if err != nil {
		if refreshErr := v.refreshKeySet(ctx); refreshErr == nil {
			v.mu.RLock()
			refreshedKeySet := v.keySet
			v.mu.RUnlock()
			parsedToken, err = jwt.Parse(
				[]byte(token),
				jwt.WithKeySet(refreshedKeySet),
				jwt.WithValidate(true),
			)
		}
	}
	if err != nil {
		return TokenClaims{}, fmt.Errorf("token verification failed: %w", err)
	}


	fmt.Println(parsedToken.Issuer(), parsedToken.Audience())
	if normalizeIssuer(parsedToken.Issuer()) != normalizeIssuer(v.issuer) {
		return TokenClaims{}, errors.New("invalid token issuer")
	}
	if !audienceMatches(parsedToken.Audience(), v.audiences) {
		return TokenClaims{}, errors.New("invalid token audience")
	}

	var scope string
	if rawScope, ok := parsedToken.Get("scope"); ok {
		scope, _ = rawScope.(string)
	}

	var scp any
	if rawScp, ok := parsedToken.Get("scp"); ok {
		scp = rawScp
	}

	scopes := extractScopes(scope, scp)
	return TokenClaims{Scopes: scopes}, nil
}

func (v *OIDCValidator) getKeySet(ctx context.Context) (jwk.Set, error) {
	v.mu.RLock()
	keySet := v.keySet
	lastFetch := v.lastFetch
	v.mu.RUnlock()

	if keySet != nil && time.Since(lastFetch) < v.refreshInterval {
		return keySet, nil
	}

	if err := v.refreshKeySet(ctx); err != nil {
		if keySet != nil {
			return keySet, nil
		}
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.keySet == nil {
		return nil, errors.New("jwks keyset unavailable")
	}
	return v.keySet, nil
}

func (v *OIDCValidator) refreshKeySet(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURI, nil)
	if err != nil {
		return fmt.Errorf("failed to create jwks request: %w", err)
	}

	res, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch jwks: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned status %d", res.StatusCode)
	}
	set, err := jwk.ParseReader(res.Body)
	if err != nil {
		return fmt.Errorf("failed to parse jwks: %w", err)
	}
	if set.Len() == 0 {
		return errors.New("jwks does not contain any usable keys")
	}

	v.mu.Lock()
	v.keySet = set
	v.lastFetch = time.Now()
	v.mu.Unlock()

	return nil
}

func normalizeIssuer(issuer string) string {
	return strings.TrimRight(strings.TrimSpace(issuer), "/")
}

func audienceMatches(claimAud any, expected map[string]struct{}) bool {
	switch aud := claimAud.(type) {
	case string:
		_, exists := expected[aud]
		return exists
	case []string:
		for _, item := range aud {
			if _, exists := expected[item]; exists {
				return true
			}
		}
	case []any:
		for _, item := range aud {
			itemStr, ok := item.(string)
			if !ok {
				continue
			}
			if _, exists := expected[itemStr]; exists {
				return true
			}
		}
	}
	return false
}

func extractScopes(scope string, scp any) []string {
	result := make([]string, 0)
	for _, item := range strings.Fields(scope) {
		result = append(result, item)
	}

	switch typed := scp.(type) {
	case string:
		for _, item := range strings.Fields(typed) {
			result = append(result, item)
		}
	case []any:
		for _, item := range typed {
			value, ok := item.(string)
			if ok && strings.TrimSpace(value) != "" {
				result = append(result, strings.TrimSpace(value))
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
