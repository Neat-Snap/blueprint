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

func (rc *Redis) AllowOncePer(ctx context.Context, purpose, email string, period time.Duration) (ok bool, ttl time.Duration, err error) {
	key := rc.Key(purpose+"_once", strings.ToLower(strings.TrimSpace(email)))
	set, err := rc.R.SetNX(ctx, key, 1, period).Result()
	if err != nil {
		return false, 0, err
	}
	if set {
		return true, period, nil
	}
	t, _ := rc.R.TTL(ctx, key).Result()
	return false, t, nil
}

var (
	ErrNotFound     = errors.New("code not found")
	ErrExpired      = errors.New("code expired")
	ErrConsumed     = errors.New("code already used")
	ErrTooMany      = errors.New("too many attempts")
	ErrMismatch     = errors.New("invalid code")
	ErrLimitReached = errors.New("too many attempts, try again later")
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

func (rc *Redis) resendKey(purpose, email string) string {
	return rc.Key(purpose+"_resend", strings.ToLower(strings.TrimSpace(email)))
}

func (rc *Redis) IncrementResend(ctx context.Context, purpose, email string, window time.Duration) (count int64, ttl time.Duration, err error) {
	key := rc.resendKey(purpose, email)

	pipe := rc.R.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	if _, err = pipe.Exec(ctx); err != nil {
		return 0, 0, err
	}

	count = incr.Val()
	ttl, _ = rc.R.TTL(ctx, key).Result()
	return count, ttl, nil
}

func (rc *Redis) Create(ctx context.Context, secret []byte, purpose, email string, codeLen int, ttl time.Duration, maxAttempts int) (id, code string, err error) {
	id = uuid.NewString()
	code, err = randomDigits(codeLen)
	if err != nil {
		return "", "", err
	}
	mac := codeMAC(secret, id, code)
	key := rc.Key(purpose, id)
	keyIdx := rc.Key(purpose+"_idx", strings.ToLower(strings.TrimSpace(email)))

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
	if err := rc.R.Set(ctx, keyIdx, id, ttl).Err(); err != nil {
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
	if em := vals["email"]; em != "" {
		_ = rc.R.Del(ctx, rc.Key(purpose+"_idx", em)).Err()
	}
	return vals["email"], nil
}

func (rc *Redis) GetIdByEmail(ctx context.Context, purpose, email string) (string, error) {
	id, err := rc.R.Get(ctx, rc.Key(purpose+"_idx", strings.ToLower(strings.TrimSpace(email)))).Result()
	if err == redis.Nil {
		return "", ErrNotFound
	}
	return id, err
}
