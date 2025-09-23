package workosclient

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/securecookie"
	"github.com/workos/workos-go/v5/pkg/usermanagement"
)

const (
	AccessCookieName  = "workos_access_token"
	RefreshCookieName = "workos_refresh_token"
	oauthStateName    = "workos_oauth_state"
)

type Client struct {
	clientID      string
	secure        bool
	sameSite      http.SameSite
	cookie        *securecookie.SecureCookie
	refreshMaxAge int
	jwksURL       string

	keyfuncOnce sync.Once
	keyfunc     keyfunc.Keyfunc
	keyfuncErr  error
}

type SessionState struct {
	RefreshToken string `json:"refresh_token"`
	SessionID    string `json:"session_id"`
}

type OAuthState struct {
	Redirect string `json:"redirect,omitempty"`
}

type AuthorizationParams struct {
	ConnectionID string
	Provider     string
	RedirectURI  string
	State        string
}

func New(cfg config.Config) (*Client, error) {
	if strings.TrimSpace(cfg.WORKOS_API_KEY) == "" {
		return nil, errors.New("WORKOS_API_KEY is not configured")
	}
	if strings.TrimSpace(cfg.WORKOS_CLIENT_ID) == "" {
		return nil, errors.New("WORKOS_CLIENT_ID is not configured")
	}
	if len(cfg.WORKOS_COOKIE_SECRET) < 16 {
		return nil, errors.New("WORKOS_COOKIE_SECRET must be at least 16 characters")
	}

	usermanagement.SetAPIKey(cfg.WORKOS_API_KEY)
	if usermanagement.DefaultClient.HTTPClient == nil {
		usermanagement.DefaultClient.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	} else {
		usermanagement.DefaultClient.HTTPClient.Timeout = 10 * time.Second
	}

	jwksURL, err := usermanagement.GetJWKSURL(cfg.WORKOS_CLIENT_ID)
	if err != nil {
		return nil, fmt.Errorf("failed to build JWKS URL: %w", err)
	}

	hashKey := sha256.Sum256([]byte(cfg.WORKOS_COOKIE_SECRET + "-hash"))
	blockKey := sha256.Sum256([]byte(cfg.WORKOS_COOKIE_SECRET + "-block"))

	sc := securecookie.New(hashKey[:], blockKey[:])
	sc.SetSerializer(securecookie.JSONEncoder{})

	return &Client{
		clientID:      cfg.WORKOS_CLIENT_ID,
		secure:        cfg.Env == "prod",
		sameSite:      http.SameSiteLaxMode,
		cookie:        sc,
		refreshMaxAge: 30 * 24 * 3600,
		jwksURL:       jwksURL.String(),
	}, nil
}

func (c *Client) getKeyfunc() (keyfunc.Keyfunc, error) {
	c.keyfuncOnce.Do(func() {
		override := keyfunc.Override{
			HTTPTimeout:       5 * time.Second,
			RefreshInterval:   time.Hour,
			ValidationSkipAll: false,
		}
		kf, err := keyfunc.NewDefaultOverrideCtx(context.Background(), []string{c.jwksURL}, override)
		if err != nil {
			c.keyfuncErr = fmt.Errorf("failed to initialise JWKS provider: %w", err)
			return
		}
		c.keyfunc = kf
	})
	if c.keyfuncErr != nil {
		return nil, c.keyfuncErr
	}
	return c.keyfunc, nil
}

func (c *Client) ParseAndValidateAccessToken(ctx context.Context, token string) (jwt.MapClaims, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("token is empty")
	}

	keyf, err := c.getKeyfunc()
	if err != nil {
		return nil, err
	}

	claims := jwt.MapClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	parsed, err := parser.ParseWithClaims(token, claims, keyf.KeyfuncCtx(ctx))
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("token is invalid")
	}

	issRaw, _ := claims["iss"].(string)
	issuer := strings.TrimSuffix(strings.TrimSpace(issRaw), "/")
	if issuer == "" {
		return nil, errors.New("token missing issuer")
	}

	if !validIssuer(issuer) {
		return nil, fmt.Errorf("token has invalid issuer")
	}

	if aud, ok := claims["aud"]; ok {
		if !includesAudience(aud, c.clientID) {
			return nil, fmt.Errorf("token has unexpected audience")
		}
	}

	return claims, nil
}

func validIssuer(issuer string) bool {
	parsed, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Host))
	if host == "" {
		return false
	}
	switch host {
	case "api.workos.com", "auth.workos.com":
		return true
	default:
		return false
	}
}

func includesAudience(value any, expected string) bool {
	switch v := value.(type) {
	case string:
		return strings.EqualFold(v, expected)
	case []string:
		for _, item := range v {
			if strings.EqualFold(item, expected) {
				return true
			}
		}
		return false
	case []any:
		for _, item := range v {
			if includesAudience(item, expected) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (c *Client) EncodeSession(state SessionState) (string, error) {
	if state.RefreshToken == "" || state.SessionID == "" {
		return "", errors.New("invalid session state")
	}
	return c.cookie.Encode(RefreshCookieName, state)
}

func (c *Client) DecodeSession(value string) (SessionState, error) {
	var state SessionState
	if err := c.cookie.Decode(RefreshCookieName, value, &state); err != nil {
		return SessionState{}, err
	}
	if state.RefreshToken == "" || state.SessionID == "" {
		return SessionState{}, errors.New("session cookie missing data")
	}
	return state, nil
}

func (c *Client) SessionFromRequest(r *http.Request) (SessionState, error) {
	cookie, err := r.Cookie(RefreshCookieName)
	if err != nil {
		return SessionState{}, err
	}
	return c.DecodeSession(cookie.Value)
}

func (c *Client) EncodeOAuthState(state OAuthState) (string, error) {
	return c.cookie.Encode(oauthStateName, state)
}

func (c *Client) DecodeOAuthState(value string) (OAuthState, error) {
	var state OAuthState
	if err := c.cookie.Decode(oauthStateName, value, &state); err != nil {
		return OAuthState{}, err
	}
	return state, nil
}

func (c *Client) AccessTokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(AccessCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value
	}
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	return ""
}

func (c *Client) SetSessionCookies(w http.ResponseWriter, accessToken string, expires time.Time, state SessionState) error {
	if strings.TrimSpace(accessToken) == "" {
		return errors.New("access token is empty")
	}
	encoded, err := c.EncodeSession(state)
	if err != nil {
		return err
	}

	ttl := time.Until(expires)
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	maxAge := int(ttl.Seconds())

	http.SetCookie(w, &http.Cookie{
		Name:     AccessCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: c.sameSite,
		Expires:  expires,
		MaxAge:   maxAge,
	})

	refreshExpires := time.Now().Add(time.Duration(c.refreshMaxAge) * time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: c.sameSite,
		MaxAge:   c.refreshMaxAge,
		Expires:  refreshExpires,
	})
	return nil
}

func (c *Client) ClearSessionCookies(w http.ResponseWriter) {
	expired := time.Unix(0, 0)
	cookies := []*http.Cookie{
		{
			Name:     AccessCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   c.secure,
			SameSite: c.sameSite,
			Expires:  expired,
			MaxAge:   -1,
		},
		{
			Name:     RefreshCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   c.secure,
			SameSite: c.sameSite,
			Expires:  expired,
			MaxAge:   -1,
		},
	}
	for _, cookie := range cookies {
		http.SetCookie(w, cookie)
	}
}

func (c *Client) AuthenticateWithPassword(ctx context.Context, email, password, ip, userAgent string) (usermanagement.AuthenticateResponse, error) {
	opts := usermanagement.AuthenticateWithPasswordOpts{
		ClientID:  c.clientID,
		Email:     email,
		Password:  password,
		IPAddress: ip,
		UserAgent: userAgent,
	}
	return usermanagement.AuthenticateWithPassword(ctx, opts)
}

func (c *Client) AuthenticateWithCode(ctx context.Context, code, ip, userAgent string) (usermanagement.AuthenticateResponse, error) {
	opts := usermanagement.AuthenticateWithCodeOpts{
		ClientID:  c.clientID,
		Code:      code,
		IPAddress: ip,
		UserAgent: userAgent,
	}
	return usermanagement.AuthenticateWithCode(ctx, opts)
}

func (c *Client) AuthenticateWithRefreshToken(ctx context.Context, refreshToken, ip, userAgent string) (usermanagement.RefreshAuthenticationResponse, error) {
	opts := usermanagement.AuthenticateWithRefreshTokenOpts{
		ClientID:     c.clientID,
		RefreshToken: refreshToken,
		IPAddress:    ip,
		UserAgent:    userAgent,
	}
	return usermanagement.AuthenticateWithRefreshToken(ctx, opts)
}

func (c *Client) CreateUser(ctx context.Context, opts usermanagement.CreateUserOpts) (usermanagement.User, error) {
	return usermanagement.CreateUser(ctx, opts)
}

func (c *Client) SendVerificationEmail(ctx context.Context, userID string) (usermanagement.UserResponse, error) {
	return usermanagement.SendVerificationEmail(ctx, usermanagement.SendVerificationEmailOpts{User: userID})
}

func (c *Client) VerifyEmail(ctx context.Context, userID, code string) (usermanagement.UserResponse, error) {
	return usermanagement.VerifyEmail(ctx, usermanagement.VerifyEmailOpts{User: userID, Code: code})
}

func (c *Client) GetUser(ctx context.Context, userID string) (usermanagement.User, error) {
	return usermanagement.GetUser(ctx, usermanagement.GetUserOpts{User: userID})
}

func (c *Client) AuthorizationURL(params AuthorizationParams) (string, error) {
	opts := usermanagement.GetAuthorizationURLOpts{
		ClientID: c.clientID,
		State:    params.State,
	}
	if params.RedirectURI != "" {
		opts.RedirectURI = params.RedirectURI
	}
	if params.ConnectionID != "" {
		opts.ConnectionID = params.ConnectionID
	}
	if params.Provider != "" {
		opts.Provider = params.Provider
	}
	url, err := usermanagement.GetAuthorizationURL(opts)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (c *Client) GetLogoutURL(sessionID, returnTo string) (string, error) {
	opts := usermanagement.GetLogoutURLOpts{SessionID: sessionID, ReturnTo: returnTo}
	url, err := usermanagement.GetLogoutURL(opts)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (c *Client) RevokeSession(ctx context.Context, sessionID string) error {
	return usermanagement.RevokeSession(ctx, usermanagement.RevokeSessionOpts{SessionID: sessionID})
}
