package email

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type Redis struct {
	R   *redis.Client
	Key func(purpose, id string) string
}

var (
	ErrNotFound = errors.New("code not found")
	ErrExpired  = errors.New("code expired")
	ErrConsumed = errors.New("code already used")
	ErrTooMany  = errors.New("too many attempts")
	ErrMismatch = errors.New("invalid code")
)

func randomDigits(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	digits := make([]byte, n)
	for i := 0; i < n; i++ {
		digits[i] = '0' + (b[i] % 10)
	}
	return string(digits), nil
}

func codeMAC(secret []byte, id, code string) []byte {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(id))
	m.Write([]byte(":"))
	m.Write([]byte(code))
	return m.Sum(nil)
}

func (rc *Redis) Create(ctx context.Context, secret []byte, purpose, email string, codeLen int, ttl time.Duration, maxAttempts int) (id, code string, err error) {
	id = uuid.NewString()
	code, err = randomDigits(codeLen)
	if err != nil {
		return "", "", err
	}
	mac := codeMAC(secret, id, code)
	key := rc.Key(purpose, id)
	err = rc.R.HSet(ctx, key, map[string]interface{}{
		"email":    strings.ToLower(strings.TrimSpace(email)),
		"mac":      hex.EncodeToString(mac),
		"len":      codeLen,
		"max":      maxAttempts,
		"tries":    0,
		"consumed": 0,
	}).Err()
	if err != nil {
		return "", "", err
	}
	if err := rc.R.Expire(ctx, key, ttl).Err(); err != nil {
		return "", "", err
	}
	return id, code, nil
}

func (rc *Redis) Verify(ctx context.Context, secret []byte, purpose, id, code string) (email string, err error) {
	key := rc.Key(purpose, id)
	vals, err := rc.R.HGetAll(ctx, key).Result()
	if err != nil {
		return "", err
	}
	if len(vals) == 0 {
		return "", ErrNotFound
	}
	if vals["consumed"] == "1" {
		return "", ErrConsumed
	}

	tries, err := rc.R.HIncrBy(ctx, key, "tries", 1).Result()
	if err != nil {
		return "", err
	}
	max, _ := strconv.ParseInt(vals["max"], 10, 64)
	if tries > max {
		return "", ErrTooMany
	}

	got := codeMAC(secret, id, strings.TrimSpace(code))
	want, _ := hex.DecodeString(vals["mac"])
	if !hmac.Equal(got, want) {
		return "", ErrMismatch
	}

	if err := rc.R.HSet(ctx, key, "consumed", 1).Err(); err != nil {
		return "", err
	}
	return vals["email"], nil
}
