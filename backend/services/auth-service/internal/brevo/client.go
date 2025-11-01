package brevo

import (
	"context"
	"fmt"
	"net/http"
	"time"

	brevosdk "github.com/getbrevo/brevo-go/lib" 
)

// Client defines the Brevo email client interface
type Client interface {
	SendTransactionalEmail(ctx context.Context, toEmail, toName, subject, htmlContent string) error
	SendVerificationEmail(ctx context.Context, toEmail, toName, verificationCode string) error
}

// brevoClient implements the Client interface using Brevo SDK
type brevoClient struct {
	apiKey    string
	fromEmail string
	fromName  string
	apiClient *brevosdk.APIClient // This is correct, as APIClient is usually part of the main client struct
}

// NewClient creates a new Brevo client
func NewClient(apiKey, fromEmail, fromName string) Client {
	cfg := brevosdk.NewConfiguration() // Correct: NewConfiguration is a function in the brevosdk package
	cfg.AddDefaultHeader("api-key", apiKey)
	cfg.AddDefaultHeader("partner-key", apiKey) // Some Brevo endpoints also use partner-key

	// Configure HTTP client with timeout
	httpClient := &http.Client{Timeout: 10 * time.Second}
	cfg.HTTPClient = httpClient

	return &brevoClient{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		apiClient: brevosdk.NewAPIClient(cfg), // Correct
	}
}

// SendTransactionalEmail sends a generic transactional email
func (bc *brevoClient) SendTransactionalEmail(ctx context.Context, toEmail, toName, subject, htmlContent string) error {
	// Corrected: Refer directly to brevosdk for types
	sender := brevosdk.SendSmtpEmailSender{
		Email: bc.fromEmail,
		Name:  bc.fromName,
	}
	to := []brevosdk.SendSmtpEmailTo{{Email: toEmail, Name: toName}}

	email := brevosdk.SendSmtpEmail{
		Sender:      &sender,
		To:          to,
		Subject:     subject,
		HtmlContent: htmlContent,
	}

	_, _, err := bc.apiClient.TransactionalEmailsApi.SendTransacEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to send transactional email via Brevo: %w", err)
	}
	return nil
}

// SendVerificationEmail sends an email with a verification code.
func (bc *brevoClient) SendVerificationEmail(ctx context.Context, toEmail, toName, verificationCode string) error {
	subject := "Verify Your Chat App Email"
	htmlContent := fmt.Sprintf(`
		<p>Hello %s,</p>
		<p>Thank you for registering with Chat App. Please use the following code to verify your email address:</p>
		<h2>%s</h2>
		<p>This code is valid for <strong>5 minutes</strong>.</p>
		<p>If you did not request this, please ignore this email.</p>
		<p>Best regards,</p>
		<p>The Chat App Team</p>
	`, toName, verificationCode)

	return bc.SendTransactionalEmail(ctx, toEmail, toName, subject, htmlContent)
}
