package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	APIKey    string
	FromEmail string
	FromName  string
}

func NewClient(apiKey, fromEmail, fromName string) *Client {
	if apiKey == "" {
		return &Client{}
	}
	return &Client{APIKey: apiKey, FromEmail: fromEmail, FromName: fromName}
}

type sendEmailReq struct {
	Sender      map[string]string   `json:"sender"`
	To          []map[string]string `json:"to"`
	Subject     string              `json:"subject"`
	HtmlContent string              `json:"htmlContent"`
}

func (c *Client) SendEmail(ctx context.Context, toEmail, subject, html string) error {
	if c.APIKey == "" {
		return nil
	}
	req := sendEmailReq{
		Sender:      map[string]string{"email": c.FromEmail, "name": c.FromName},
		To:          []map[string]string{{"email": toEmail}},
		Subject:     subject,
		HtmlContent: html,
	}
	b, _ := json.Marshal(req)
	r, _ := http.NewRequestWithContext(ctx, "POST", "https://api.brevo.com/v3/smtp/email", bytes.NewReader(b))
	r.Header.Set("api-key", c.APIKey)
	r.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("brevo status %d", resp.StatusCode)
	}
	return nil
}
