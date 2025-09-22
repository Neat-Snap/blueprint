package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/workos/workos-go/v4/pkg/usermanagement"
	"github.com/workos/workos-go/v4/pkg/workos_errors"
)

const (
	// WorkOSProvider is the identifier for WorkOS auth identities stored in the local database.
	WorkOSProvider = "workos"

	// AccessTokenCookieName is the cookie used to store the WorkOS access token.
	AccessTokenCookieName = "bp_access_token"
	// SessionIDCookieName stores the WorkOS session identifier associated with the current access token.
	SessionIDCookieName = "bp_session_id"
	// StateCookieName stores the opaque state used for WorkOS OAuth flows.
	StateCookieName = "bp_workos_state"
)

// AccessTokenClaims represents the essential fields extracted from a WorkOS access token.
type AccessTokenClaims struct {
	Subject   string
	SessionID string
	ExpiresAt time.Time
}

// WorkOSError wraps an error returned by the WorkOS SDK with HTTP metadata so handlers can respond appropriately.
type WorkOSError struct {
	Status  int
	Message string
	Err     error
}

func (e *WorkOSError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "workos error"
}

func (e *WorkOSError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// WorkOSAuthService centralises interactions with the WorkOS User Management API.
type WorkOSAuthService struct {
	logger         logger.MultiLogger
	clientID       string
	connectionID   string
	organizationID string
	redirectURI    string
}

// NewWorkOSAuthService initialises a WorkOS client for the configured project.
func NewWorkOSAuthService(cfg config.Config, log logger.MultiLogger) (*WorkOSAuthService, error) {
	if cfg.WORKOS_API_KEY == "" {
		return nil, errors.New("WORKOS_API_KEY is not configured")
	}
	if cfg.WORKOS_CLIENT_ID == "" {
		return nil, errors.New("WORKOS_CLIENT_ID is not configured")
	}
	redirectURI := strings.TrimSuffix(cfg.BACKEND_PUBLIC_URL, "/") + "/auth/callback"

	if cfg.WORKOS_CONNECTION_ID == "" && cfg.WORKOS_ORGANIZATION_ID == "" {
		return nil, errors.New("either WORKOS_CONNECTION_ID or WORKOS_ORGANIZATION_ID must be configured")
	}

	usermanagement.SetAPIKey(cfg.WORKOS_API_KEY)

	svc := &WorkOSAuthService{
		logger:         log,
		clientID:       cfg.WORKOS_CLIENT_ID,
		connectionID:   cfg.WORKOS_CONNECTION_ID,
		organizationID: cfg.WORKOS_ORGANIZATION_ID,
		redirectURI:    redirectURI,
	}
	return svc, nil
}

// RedirectURI returns the callback URL registered with WorkOS.
func (s *WorkOSAuthService) RedirectURI() string {
	return s.redirectURI
}

// ClientID returns the configured WorkOS client identifier.
func (s *WorkOSAuthService) ClientID() string {
	return s.clientID
}

// GenerateState produces a cryptographically secure state parameter for OAuth flows.
func (s *WorkOSAuthService) GenerateState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// AuthorizationURL generates the WorkOS authorization URL for the provided hint.
func (s *WorkOSAuthService) AuthorizationURL(state string, hint usermanagement.ScreenHint) (*url.URL, error) {
	opts := usermanagement.GetAuthorizationURLOpts{
		ClientID:    s.clientID,
		RedirectURI: s.redirectURI,
		State:       state,
	}
	if hint != "" {
		opts.ScreenHint = hint
	}
	if s.connectionID != "" {
		opts.ConnectionID = s.connectionID
	} else if s.organizationID != "" {
		opts.OrganizationID = s.organizationID
	}
	return usermanagement.GetAuthorizationURL(opts)
}

// AuthenticateWithCode exchanges an authorization code for WorkOS session tokens.
func (s *WorkOSAuthService) AuthenticateWithCode(ctx context.Context, code, ip, userAgent string) (usermanagement.AuthenticateResponse, error) {
	resp, err := usermanagement.AuthenticateWithCode(ctx, usermanagement.AuthenticateWithCodeOpts{
		ClientID:  s.clientID,
		Code:      code,
		IPAddress: strings.TrimSpace(ip),
		UserAgent: strings.TrimSpace(userAgent),
	})
	if err != nil {
		return usermanagement.AuthenticateResponse{}, s.wrapError(err, "failed to authenticate with WorkOS")
	}
	return resp, nil
}

// AuthenticateWithRefreshToken refreshes a WorkOS session using the provided refresh token.
func (s *WorkOSAuthService) AuthenticateWithRefreshToken(ctx context.Context, refreshToken, ip, userAgent string) (usermanagement.RefreshAuthenticationResponse, error) {
	resp, err := usermanagement.AuthenticateWithRefreshToken(ctx, usermanagement.AuthenticateWithRefreshTokenOpts{
		ClientID:       s.clientID,
		RefreshToken:   refreshToken,
		OrganizationID: s.organizationID,
		IPAddress:      strings.TrimSpace(ip),
		UserAgent:      strings.TrimSpace(userAgent),
	})
	if err != nil {
		return usermanagement.RefreshAuthenticationResponse{}, s.wrapError(err, "failed to refresh WorkOS session")
	}
	return resp, nil
}

// SendVerificationEmail triggers WorkOS to send a verification email to the given user.
func (s *WorkOSAuthService) SendVerificationEmail(ctx context.Context, userID string) error {
	_, err := usermanagement.SendVerificationEmail(ctx, usermanagement.SendVerificationEmailOpts{User: userID})
	if err != nil {
		return s.wrapError(err, "failed to send verification email")
	}
	return nil
}

// CreatePasswordReset requests WorkOS to send a password reset email to the given address.
func (s *WorkOSAuthService) CreatePasswordReset(ctx context.Context, email string) (usermanagement.PasswordReset, error) {
	reset, err := usermanagement.CreatePasswordReset(ctx, usermanagement.CreatePasswordResetOpts{Email: email})
	if err != nil {
		return usermanagement.PasswordReset{}, s.wrapError(err, "failed to create password reset")
	}
	return reset, nil
}

// ResetPassword finalises a password reset using the WorkOS token.
func (s *WorkOSAuthService) ResetPassword(ctx context.Context, token, newPassword string) (usermanagement.UserResponse, error) {
	user, err := usermanagement.ResetPassword(ctx, usermanagement.ResetPasswordOpts{Token: token, NewPassword: newPassword})
	if err != nil {
		return usermanagement.UserResponse{}, s.wrapError(err, "failed to reset password")
	}
	return user, nil
}

// RevokeSession revokes a WorkOS session by session identifier.
func (s *WorkOSAuthService) RevokeSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return &WorkOSError{Status: http.StatusBadRequest, Message: "missing session identifier"}
	}
	if err := usermanagement.RevokeSession(ctx, usermanagement.RevokeSessionOpts{SessionID: sessionID}); err != nil {
		return s.wrapError(err, "failed to revoke WorkOS session")
	}
	return nil
}

// ListUsersByEmail fetches WorkOS users filtered by email address.
func (s *WorkOSAuthService) ListUsersByEmail(ctx context.Context, email string) (usermanagement.ListUsersResponse, error) {
	resp, err := usermanagement.ListUsers(ctx, usermanagement.ListUsersOpts{Email: email, Limit: 1})
	if err != nil {
		return usermanagement.ListUsersResponse{}, s.wrapError(err, "failed to look up user by email")
	}
	return resp, nil
}

// GetUser fetches a WorkOS user by identifier.
func (s *WorkOSAuthService) GetUser(ctx context.Context, id string) (usermanagement.User, error) {
	user, err := usermanagement.GetUser(ctx, usermanagement.GetUserOpts{User: id})
	if err != nil {
		return usermanagement.User{}, s.wrapError(err, "failed to fetch user")
	}
	return user, nil
}

// ParseAccessToken extracts the subject, session ID, and expiry from a WorkOS access token.
func (s *WorkOSAuthService) ParseAccessToken(token string) (AccessTokenClaims, error) {
	var claims AccessTokenClaims
	if strings.TrimSpace(token) == "" {
		return claims, errors.New("empty access token")
	}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	jwtClaims := jwt.MapClaims{}
	if _, _, err := parser.ParseUnverified(token, jwtClaims); err != nil {
		return claims, err
	}
	if sub, ok := jwtClaims["sub"].(string); ok {
		claims.Subject = sub
	}
	if sid, ok := jwtClaims["sid"].(string); ok {
		claims.SessionID = sid
	}
	if exp := extractExpiry(jwtClaims["exp"]); !exp.IsZero() {
		claims.ExpiresAt = exp
	}
	if claims.Subject == "" {
		return claims, errors.New("missing subject in access token")
	}
	return claims, nil
}

func extractExpiry(value interface{}) time.Time {
	switch v := value.(type) {
	case float64:
		return time.Unix(int64(v), 0)
	case int64:
		return time.Unix(v, 0)
	case int32:
		return time.Unix(int64(v), 0)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return time.Unix(i, 0)
		}
	case string:
		if v == "" {
			return time.Time{}
		}
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Unix(i, 0)
		}
	}
	return time.Time{}
}

func (s *WorkOSAuthService) wrapError(err error, fallback string) *WorkOSError {
	if err == nil {
		return nil
	}
	var httpErr workos_errors.HTTPError
	if errors.As(err, &httpErr) {
		message := strings.TrimSpace(httpErr.Message)
		if message == "" {
			message = fallback
		}
		return &WorkOSError{
			Status:  httpErr.Code,
			Message: message,
			Err:     err,
		}
	}
	return &WorkOSError{Status: http.StatusBadGateway, Message: fallback, Err: err}
}
