package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrEmailTaken       = errors.New("email already in use")
	ErrOAuthOnlyAccount = errors.New("oauth only account")
)

func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return constraint == "" || pgErr.ConstraintName == constraint
	}
	return false
}

func GenerateJWT(secret []byte, email string, iss string, aud string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"iat":   now.Unix(),
		"iss":   iss,
		"aud":   aud,
		"exp":   now.Add(21 * 24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func DecodeJWT(secret []byte, tokenStr string, iss string, aud string) (string, error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		if err == nil {
			err = fmt.Errorf("invalid token")
		}
		return "", err
	}

	now := time.Now().Unix()
	if expVal, ok := claims["exp"]; ok {
		switch v := expVal.(type) {
		case float64:
			if int64(v) < now {
				return "", fmt.Errorf("token expired")
			}
		case json.Number:
			if n, err := v.Int64(); err == nil && n < now {
				return "", fmt.Errorf("token expired")
			}
		}
	}

	if issVal, ok := claims["iss"].(string); !ok || issVal != iss {
		return "", fmt.Errorf("invalid issuer")
	}

	switch audVal := claims["aud"].(type) {
	case string:
		if audVal != aud {
			return "", fmt.Errorf("invalid audience")
		}
	case []interface{}:
		found := false
		for _, a := range audVal {
			if s, ok := a.(string); ok && s == aud {
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("invalid audience")
		}
	default:
		return "", fmt.Errorf("invalid audience")
	}

	sub, ok := claims["email"]
	if !ok {
		return "", fmt.Errorf("the email subject was not found in jwt")
	}
	s, ok := sub.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("invalid email claim")
	}
	return s, nil
}
