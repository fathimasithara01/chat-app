package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"go.uber.org/zap"
)

// EmailNotifier sends transactional emails via Brevo (Sendinblue) HTTP API v3
type EmailNotifier struct {
	APIKey      string
	SenderEmail string
	SenderName  string
	client      *http.Client
	logger      *zap.SugaredLogger
	templates   map[string]*template.Template
}

// NewEmailNotifier initializes notifier and pre-parses templates
func NewEmailNotifier(apiKey, senderEmail, senderName string, logger *zap.SugaredLogger) *EmailNotifier {
	templates := map[string]*template.Template{}
	// parse templates from files (templates folder)
	t1 := template.Must(template.ParseFiles("internal/templates/email_welcome.html"))
	t2 := template.Must(template.ParseFiles("internal/templates/email_chat_message.html"))
	templates["welcome"] = t1
	templates["chat_message"] = t2

	return &EmailNotifier{
		APIKey:      apiKey,
		SenderEmail: senderEmail,
		SenderName:  senderName,
		client:      &http.Client{Timeout: 10 * time.Second},
		logger:      logger,
		templates:   templates,
	}
}

// SendTemplateEmail sends an email using a templateKey and data
func (e *EmailNotifier) SendTemplateEmail(ctx context.Context, toEmail, subject, templateKey string, data any) error {
	tpl, ok := e.templates[templateKey]
	if !ok {
		return fmt.Errorf("template %q not found", templateKey)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return err
	}
	// Build Brevo request payload
	payload := map[string]any{
		"sender":      map[string]string{"name": e.SenderName, "email": e.SenderEmail},
		"to":          []map[string]string{{"email": toEmail}},
		"subject":     subject,
		"htmlContent": buf.String(),
	}
	// encode
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.brevo.com/v3/smtp/email", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", e.APIKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		e.logger.Infof("email sent to %s subject=%s", toEmail, subject)
		return nil
	}
	e.logger.Warnf("brevo send failed status=%d", resp.StatusCode)
	return fmt.Errorf("brevo send failed status=%d", resp.StatusCode)
}
