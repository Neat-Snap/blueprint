package utils

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/resend/resend-go/v2"
)

type EmailClient struct {
	apikey       string
	resendClient *resend.Client
	logger       logger.MultiLogger
}

type ConfirmationEmailVars struct {
	AppName      string
	Code         string
	ExpiresMin   int
	ActionUrl    string
	SupportEmail string
	CurrentYear  int
}

func NewEmailClient(apikey string) *EmailClient {
	return &EmailClient{
		apikey:       apikey,
		resendClient: resend.NewClient(apikey),
	}
}

func loadTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
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

func (e *EmailClient) GetTemplateFromFile(recipient string, subject string, filename string) (string, error) {
	tmpl, err := loadTemplate(filename)
	if err != nil {
		e.logger.Fatal("failed to load template", "error", err)
		return "", err
	}

	return tmpl, nil
}

func (e *EmailClient) SendConfirmationEmail(recipient string, subject string, vars ConfirmationEmailVars) (string, error) {
	tmpl, err := e.GetTemplateFromFile(recipient, subject, "confirmation_template.html")
	if err != nil {
		return "", err
	}

	html := strings.NewReplacer(
		"{{APP_NAME}}", vars.AppName,
		"{{CODE}}", vars.Code,
		"{{EXPIRES_MIN}}", fmt.Sprint(vars.ExpiresMin),
		"{{ACTION_URL}}", vars.ActionUrl,
		"{{SUPPORT_EMAIL}}", vars.SupportEmail,
		"{{CURRENT_YEAR}}", fmt.Sprint(time.Now().Year()),
	).Replace(tmpl)

	return e.SendEmail(recipient, subject, html)
}
