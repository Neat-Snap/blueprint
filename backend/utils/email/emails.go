package email

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/go-redis/redis/v8"
	"github.com/resend/resend-go/v2"
)

type EmailClient struct {
	apikey       string
	resendClient *resend.Client
	logger       logger.MultiLogger
	R            *Redis
	Config       config.Config
}

var (
	VerifyPurpose        = "email_verify"
	ResetPasswordPurpose = "password_reset"
)

func NewEmailClient(cfg config.Config, logger logger.MultiLogger) *EmailClient {
	return &EmailClient{
		apikey:       cfg.RESEND_API_KEY,
		resendClient: resend.NewClient(cfg.RESEND_API_KEY),
		logger:       logger,
		R: &Redis{
			R: redis.NewClient(&redis.Options{
				Addr:     cfg.REDIS_HOST + ":" + cfg.REDIS_PORT,
				Password: cfg.REDIS_PASS,
				DB:       cfg.REDIS_DB,
			}),
			Key: func(purpose, id string) string { return "verif:" + purpose + ":" + id },
		},
		Config: cfg,
	}
}

func loadTemplate(filename string) (string, error) {
	_, src, _, _ := runtime.Caller(0) // path to emails.go
	base := filepath.Dir(src)
	data, err := os.ReadFile(filepath.Join(base, filename))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (e *EmailClient) SendEmail(recipient string, subject string, body string) (string, error) {
	params := &resend.SendEmailRequest{
		From:    "Devenv <onboarding@resend.dev>",
		To:      []string{recipient},
		Html:    body,
		Subject: subject,
	}

	sent, err := e.resendClient.Emails.Send(params)
	if err != nil {
		e.logger.Error("failed to send email", "error", err)
		return "", err
	}
	return sent.Id, nil
}

func (e *EmailClient) GetTemplateFromFile(filename string) (string, error) {
	tmpl, err := loadTemplate(filename)
	if err != nil {
		e.logger.Fatal("failed to load template", "error", err)
		return "", err
	}

	return tmpl, nil
}

func (e *EmailClient) buildActionUrl(id, code string) string {
	return e.Config.APP_URL + "/auth/verify?cid=" + id + "&code=" + code
}

func (e *EmailClient) buildResetPasswordUrl(id, code string) string {
	return e.Config.APP_URL + "/auth/reset-password?cid=" + id + "&code=" + code
}

func (e *EmailClient) SendConfirmationEmail(recipient string, subject string, expiresMin int) (string, error) {
	tmpl, err := e.GetTemplateFromFile("confirmation_template.html")
	if err != nil {
		return "", err
	}

	id, code, err := e.R.Create(context.Background(), []byte(e.Config.REDIS_SECRET), VerifyPurpose, recipient, 6, time.Duration(expiresMin)*time.Minute, 6)
	if err != nil {
		return "", err
	}

	html := strings.NewReplacer(
		"{{APP_NAME}}", e.Config.APP_NAME,
		"{{CODE}}", code,
		"{{EXPIRES_MIN}}", fmt.Sprint(expiresMin),
		"{{ACTION_URL}}", e.buildActionUrl(id, code),
		"{{SUPPORT_EMAIL}}", e.Config.SUPPORT_EMAIL,
		"{{CURRENT_YEAR}}", fmt.Sprint(time.Now().Year()),
	).Replace(tmpl)

	_, err = e.SendEmail(recipient, subject, html)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (e *EmailClient) SendResetPasswordEmail(recipient string, subject string, expiresMin int) (string, error) {
	tmpl, err := e.GetTemplateFromFile("reset_password_template.html")
	if err != nil {
		return "", err
	}

	id, code, err := e.R.Create(context.Background(), []byte(e.Config.REDIS_SECRET), ResetPasswordPurpose, recipient, 6, time.Duration(expiresMin)*time.Minute, 1)
	if err != nil {
		return "", err
	}

	html := strings.NewReplacer(
		"{{APP_NAME}}", e.Config.APP_NAME,
		"{{EXPIRES_MIN}}", fmt.Sprint(expiresMin),
		"{{RESET_URL}}", e.buildResetPasswordUrl(id, code),
		"{{SUPPORT_EMAIL}}", e.Config.SUPPORT_EMAIL,
		"{{CURRENT_YEAR}}", fmt.Sprint(time.Now().Year()),
	).Replace(tmpl)

	_, err = e.SendEmail(recipient, subject, html)
	if err != nil {
		return "", err
	}

	return id, nil
}
