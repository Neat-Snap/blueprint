package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/argon2"
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

type argonParams struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	SaltLen int
	KeyLen  uint32
}

var DefaultArgon = argonParams{
	Time:    2,
	Memory:  64 * 1024,
	Threads: 1,
	SaltLen: 16,
	KeyLen:  32,
}

func HashSecret(value string, p argonParams) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("empty secret")
	}
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(value), salt, p.Time, p.Memory, p.Threads, p.KeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.Memory, p.Time, p.Threads, b64Salt, b64Key), nil
}

func GenerateJWT(secret []byte, email string, iss string, aud string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(21 * 24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func DecodeJWT(secret []byte, tokenStr string) (string, error) {
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
