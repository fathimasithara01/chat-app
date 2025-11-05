package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const brevoAPIURL = "https://api.brevo.com/v3/smtp/email"

// Client represents the Brevo (formerly Sendinblue) API client.
type Client struct {
	APIKey     string
	FromEmail  string
	FromName   string
	httpClient *http.Client // Custom HTTP client for requests
	configured bool         // Flag to indicate if the client is properly configured
}

// NewClient creates and returns a new Brevo Client.
// It checks if the API key, from email, and from name are provided to mark it as configured.
func NewClient(apiKey, fromEmail, fromName string) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second}, // Initialize with a default timeout
	}

	if apiKey != "" && fromEmail != "" && fromName != "" {
		c.APIKey = apiKey
		c.FromEmail = fromEmail
		c.FromName = fromName
		c.configured = true
	}
	return c
}

// IsConfigured returns true if the Brevo client has been initialized with the necessary credentials.
func (c *Client) IsConfigured() bool {
	return c.configured
}

// sendEmailReq defines the structure for a Brevo send email request.
type sendEmailReq struct {
	Sender      map[string]string   `json:"sender"`
	To          []map[string]string `json:"to"`
	Subject     string              `json:"subject"`
	HtmlContent string              `json:"htmlContent"`
}

// SendEmail sends an email using the Brevo API.
// It will be a no-op if the client is not configured, returning nil.
func (c *Client) SendEmail(ctx context.Context, toEmail, subject, html string) error {
	if !c.configured {
		// Log this instead of just printing, as it's a condition where an email *might* have been expected.
		// The caller should ideally check IsConfigured() if silent skipping is not desired.
		return fmt.Errorf("brevo client not configured, email to %s for subject '%s' skipped", toEmail, subject)
	}

	if toEmail == "" || subject == "" || html == "" {
		return errors.New("toEmail, subject, and html content cannot be empty")
	}

	reqBody := sendEmailReq{
		Sender:      map[string]string{"email": c.FromEmail, "name": c.FromName},
		To:          []map[string]string{{"email": toEmail}},
		Subject:     subject,
		HtmlContent: html,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal email request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", brevoAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for Brevo: %w", err)
	}

	httpReq.Header.Set("api-key", c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("brevo send email request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorBody map[string]interface{}
		// Attempt to decode error body, but don't fail if it's unreadable
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errorBody); decodeErr != nil {
			return fmt.Errorf("brevo API error: status %d, failed to decode error body: %v", resp.StatusCode, decodeErr)
		}
		return fmt.Errorf("brevo API error: status %d, body: %v", resp.StatusCode, errorBody)
	}

	return nil
}
