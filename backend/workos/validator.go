package workos

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/workos/workos-go/v4/pkg/usermanagement"
)

const (
	defaultJWKSCacheTTL = time.Hour
	defaultIssuer       = "https://api.workos.com"
)

var (
	// ErrMissingAccessToken is returned when the access token is empty.
	ErrMissingAccessToken = errors.New("workos: access token is required")
	// ErrMissingRefreshToken is returned when a refresh token is required but missing.
	ErrMissingRefreshToken = errors.New("workos: refresh token is required")
)

// ValidatorConfig describes the configuration for creating a Validator.
type ValidatorConfig struct {
	Client   *usermanagement.Client
	ClientID string
	Issuer   string

	// HTTPClient is used to fetch the JWKS. Defaults to http.DefaultClient.
	HTTPClient *http.Client
	// CacheTTL controls how long JWKS responses are cached. Defaults to one hour.
	CacheTTL time.Duration
}

// ValidationResult represents the validated WorkOS access token claims.
type ValidationResult struct {
	AccessToken  string
	RefreshToken string

	UserID        string
	Email         string
	EmailVerified bool
	SessionID     string

	IssuedAt  time.Time
	ExpiresAt time.Time

	Refreshed bool
	Claims    jwt.MapClaims
}

// Validator validates WorkOS issued access tokens and refreshes them when expired.
type Validator struct {
	client   *usermanagement.Client
	clientID string
	issuer   string

	httpClient *http.Client
	cacheTTL   time.Duration

	jwksURL string

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// NewValidator builds a Validator instance using the provided configuration.
func NewValidator(cfg ValidatorConfig) (*Validator, error) {
	if cfg.Client == nil {
		return nil, errors.New("workos: client must not be nil")
	}
	if cfg.ClientID == "" {
		return nil, errors.New("workos: client id must not be empty")
	}

	jwksURL, err := usermanagement.GetJWKSURL(cfg.ClientID)
	if err != nil {
		return nil, fmt.Errorf("workos: resolve jwks url: %w", err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultJWKSCacheTTL
	}

	issuer := cfg.Issuer
	if issuer == "" {
		issuer = defaultIssuer
	}

	return &Validator{
		client:     cfg.Client,
		clientID:   cfg.ClientID,
		issuer:     issuer,
		httpClient: httpClient,
		cacheTTL:   cacheTTL,
		jwksURL:    jwksURL.String(),
		keys:       make(map[string]*rsa.PublicKey),
	}, nil
}

// ParseAccessToken verifies the provided access token and returns its claims.
// If the token has expired, the returned error satisfies errors.Is(err, jwt.ErrTokenExpired),
// while the returned ValidationResult still contains the parsed claims.
func (v *Validator) ParseAccessToken(ctx context.Context, token string) (*ValidationResult, error) {
	if token == "" {
		return nil, ErrMissingAccessToken
	}

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, v.keyFunc(ctx),
		jwt.WithAudience(v.clientID),
		jwt.WithIssuer(v.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			if parsed != nil {
				res, convErr := v.buildResult(token, claims)
				if convErr != nil {
					return nil, convErr
				}
				return res, err
			}
		}
		return nil, err
	}

	if !parsed.Valid {
		return nil, fmt.Errorf("workos: invalid token")
	}

	return v.buildResult(token, claims)
}

// Refresh obtains a new access token using the provided refresh token and validates it.
func (v *Validator) Refresh(ctx context.Context, refreshToken string) (*ValidationResult, error) {
	if refreshToken == "" {
		return nil, ErrMissingRefreshToken
	}

	resp, err := v.client.AuthenticateWithRefreshToken(ctx, usermanagement.AuthenticateWithRefreshTokenOpts{
		ClientID:     v.clientID,
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("workos: refresh token: %w", err)
	}

	res, err := v.ParseAccessToken(ctx, resp.AccessToken)
	if err != nil {
		return nil, err
	}

	res.AccessToken = resp.AccessToken
	res.RefreshToken = resp.RefreshToken
	res.Refreshed = true
	return res, nil
}

func (v *Validator) buildResult(token string, claims jwt.MapClaims) (*ValidationResult, error) {
	subRaw, ok := claims["sub"]
	if !ok {
		return nil, fmt.Errorf("workos: token missing sub claim")
	}
	sub, ok := subRaw.(string)
	if !ok || sub == "" {
		return nil, fmt.Errorf("workos: token has invalid sub claim")
	}

	res := &ValidationResult{
		AccessToken: token,
		UserID:      sub,
		Claims:      claims,
	}

	if sid, ok := claims["sid"].(string); ok {
		res.SessionID = sid
	}
	if email, ok := claims["email"].(string); ok {
		res.Email = email
	}
	if ev, ok := claims["email_verified"]; ok {
		switch v := ev.(type) {
		case bool:
			res.EmailVerified = v
		case string:
			res.EmailVerified = v == "true"
		}
	}

	if exp, err := parseNumericDate(claims["exp"]); err == nil {
		res.ExpiresAt = exp
	} else if err != nil {
		return nil, fmt.Errorf("workos: parse exp: %w", err)
	}
	if iat, err := parseNumericDate(claims["iat"]); err == nil {
		res.IssuedAt = iat
	}

	return res, nil
}

func (v *Validator) keyFunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if token == nil {
			return nil, fmt.Errorf("workos: nil token")
		}
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("workos: unexpected signing method %q", token.Header["alg"])
		}
		kidRaw, ok := token.Header["kid"]
		if !ok {
			return nil, fmt.Errorf("workos: token missing kid header")
		}
		kid, ok := kidRaw.(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("workos: invalid kid header")
		}

		if key := v.lookupKey(kid); key != nil {
			return key, nil
		}

		if err := v.refreshKeys(ctx, true); err != nil {
			return nil, err
		}
		if key := v.lookupKey(kid); key != nil {
			return key, nil
		}
		return nil, fmt.Errorf("workos: jwk with kid %q not found", kid)
	}
}

func (v *Validator) lookupKey(kid string) *rsa.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.keys[kid]
}

func (v *Validator) refreshKeys(ctx context.Context, force bool) error {
	v.mu.RLock()
	shouldFetch := force || time.Since(v.fetchedAt) > v.cacheTTL || len(v.keys) == 0
	v.mu.RUnlock()
	if !shouldFetch {
		return nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	if !force && time.Since(v.fetchedAt) <= v.cacheTTL && len(v.keys) > 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("workos: create jwks request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("workos: fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("workos: jwks request failed with status %s", resp.Status)
	}

	var payload jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("workos: decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, k := range payload.Keys {
		if k.Kty != "RSA" {
			continue
		}
		if k.Kid == "" {
			continue
		}
		pub, err := buildRSAPublicKey(k.N, k.E)
		if err != nil {
			return fmt.Errorf("workos: parse jwk %s: %w", k.Kid, err)
		}
		keys[k.Kid] = pub
	}

	v.keys = keys
	v.fetchedAt = time.Now()
	return nil
}

func parseNumericDate(value interface{}) (time.Time, error) {
	if value == nil {
		return time.Time{}, nil
	}
	switch v := value.(type) {
	case float64:
		return time.Unix(int64(v), 0), nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(i, 0), nil
	case int64:
		return time.Unix(v, 0), nil
	case int:
		return time.Unix(int64(v), 0), nil
	default:
		return time.Time{}, fmt.Errorf("unexpected numeric date type %T", value)
	}
}

func buildRSAPublicKey(nEnc, eEnc string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nEnc)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eEnc)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}
	if len(eBytes) == 0 {
		return nil, errors.New("invalid exponent length")
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid exponent value")
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

type jwksResponse struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Alg string `json:"alg"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}
