package utils

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
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
	Time:    1,
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
	e := NormalizeEmail(email)
	if e == "" || password == "" {
		return nil, errors.New("email and password required")
	}

	var out *db.User
	err := store.WithTx(ctx, func(tx *db.Connection) error {
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
			Name:            &name,
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
		out = u
		return nil
	})
	return out, err
}

func ResetPassword(ctx context.Context, store *db.Connection, email, password string) error {
	e := NormalizeEmail(email)
	if e == "" || password == "" {
		return errors.New("email and password required")
	}

	u, err := store.Users.ByEmail(ctx, e)
	if err != nil {
		return err
	}

	if u.PasswordCredential == nil || u.PasswordCredential.PasswordDisabled {
		return ErrOAuthOnlyAccount
	}

	hash, hashErr := HashPassword(password, DefaultArgon)
	if hashErr != nil {
		return hashErr
	}

	return store.Auth.EnsurePasswordCredential(ctx, u.ID, hash)
}

func GenerateJWT(secret []byte, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 24 * 21).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func DecodeJWT(secret []byte, token string) (string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return "", err
	}

	sub, ok := claims["email"]
	if !ok {
		return "", fmt.Errorf("the email subject was not found in jwt")
	}

	return sub.(string), nil
}
