package utils

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
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

func HashPassword(pw string, p argonParams) (string, error) {
	if pw == "" {
		return "", errors.New("empty password")
	}
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, p.Time, p.Memory, p.Threads, p.KeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.Memory, p.Time, p.Threads, b64Salt, b64Key), nil
}

func ComparePassword(pw, phc string) (bool, error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid hash format")
	}
	var mem uint32
	var time uint32
	var par uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &time, &par)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	key := argon2.IDKey([]byte(pw), salt, time, mem, par, uint32(len(want)))
	ok := subtle.ConstantTimeCompare(key, want) == 1
	return ok, nil
}

func SignUpEmailPassword(ctx context.Context, store *db.Connection, email, password, name string) (*db.User, error) {
	e, err := ValidateEmail(email)
	if err != nil {
		return nil, err
	}
	if err := ValidatePassword(password, DefaultPasswordPolicy()); err != nil {
		return nil, err
	}
	n, err := ValidateOptionalName(name)
	if err != nil {
		return nil, err
	}

	var out *db.User
	err = store.WithTx(ctx, func(tx *db.Connection) error {
		u, err := tx.Users.ByEmail(ctx, e)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		hash, hashErr := HashPassword(password, DefaultArgon)
		if hashErr != nil {
			return hashErr
		}

		if err == nil {
			// user exists already
			if u.PasswordCredential == nil || u.PasswordCredential.PasswordDisabled {
				return ErrOAuthOnlyAccount
			}
			return ErrEmailTaken
		}

		u = &db.User{
			Email:           &e,
			Name:            &n,
			EmailVerifiedAt: nil,
		}
		if err := tx.Users.Create(ctx, u); err != nil {
			// someone created same email concurrently
			if IsUniqueViolation(err, "uniq_users_email") {
				return ErrEmailTaken
			}
			return err
		}
		if err := tx.Auth.EnsurePasswordCredential(ctx, u.ID, hash); err != nil {
			return err
		}

		if err := tx.Preferences.Create(ctx, u.ID); err != nil {
			return err
		}

		out = u
		return nil
	})
	return out, err
}

func ResetPassword(ctx context.Context, store *db.Connection, email, password string) error {
	e, err := ValidateEmail(email)
	if err != nil {
		return err
	}
	if err := ValidatePassword(password, DefaultPasswordPolicy()); err != nil {
		return err
	}
	u, err := store.Users.ByEmail(ctx, e)
	if err != nil {
		return err
	}

	if u.PasswordCredential == nil || u.PasswordCredential.PasswordDisabled {
		return ErrOAuthOnlyAccount
	}

	hash, err := HashPassword(password, DefaultArgon)
	if err != nil {
		return err
	}

	return store.Auth.EnsurePasswordCredential(ctx, u.ID, hash)
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
