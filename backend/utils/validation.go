package utils

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"

	"github.com/Neat-Snap/blueprint-backend/config"
)

type PasswordPolicy struct {
	MinLength     int
	MaxLength     int
	RequireUpper  bool
	RequireLower  bool
	RequireNumber bool
	RequireSymbol bool
}

func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:     8,
		MaxLength:     128,
		RequireUpper:  true,
		RequireLower:  true,
		RequireNumber: true,
		RequireSymbol: true,
	}
}

func PolicyFromConfig(cfg config.Config) PasswordPolicy {
	p := DefaultPasswordPolicy()
	if cfg.PASSWORD_MIN_LENGTH > 0 {
		p.MinLength = cfg.PASSWORD_MIN_LENGTH
	}
	if cfg.PASSWORD_MAX_LENGTH > 0 {
		p.MaxLength = cfg.PASSWORD_MAX_LENGTH
	}
	p.RequireUpper = cfg.PASSWORD_REQUIRE_UPPER
	p.RequireLower = cfg.PASSWORD_REQUIRE_LOWER
	p.RequireNumber = cfg.PASSWORD_REQUIRE_NUMBER
	p.RequireSymbol = cfg.PASSWORD_REQUIRE_SYMBOL
	return p
}

var (
	symbolRe = regexp.MustCompile(`[!@#$%^&*()_+\-=[\]{}|;':",./<>?~]`)
)

func ValidateEmail(email string) (string, error) {
	e := NormalizeEmail(email)
	if e == "" {
		return "", errors.New("email is required")
	}
	addr, err := mail.ParseAddress(e)
	if err != nil {
		return "", errors.New("invalid email format")
	}
	if addr.Address == "" || strings.ContainsAny(e, " <>\n\r\t") && addr.Address != e {
		return "", errors.New("invalid email format")
	}
	return e, nil
}

func ValidatePassword(pw string, policy PasswordPolicy) error {
	if pw == "" {
		return errors.New("password is required")
	}
	if len(pw) < policy.MinLength {
		return errors.New("password is too short")
	}
	if policy.MaxLength > 0 && len(pw) > policy.MaxLength {
		return errors.New("password is too long")
	}
	if policy.RequireUpper && !strings.ContainsAny(pw, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return errors.New("password must contain an uppercase letter")
	}
	if policy.RequireLower && !strings.ContainsAny(pw, "abcdefghijklmnopqrstuvwxyz") {
		return errors.New("password must contain a lowercase letter")
	}
	if policy.RequireNumber && !strings.ContainsAny(pw, "0123456789") {
		return errors.New("password must contain a number")
	}
	if policy.RequireSymbol && !symbolRe.MatchString(pw) {
		return errors.New("password must contain a symbol")
	}
	return nil
}

func ValidateOptionalName(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", nil
	}
	if len(n) > 128 {
		return "", errors.New("name is too long")
	}
	return n, nil
}
