package emailJS

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const emailJSAPIURL = "https://api.emailjs.com/api/v1.0/email/send"

type Client struct {
	PublicKey  string
	PrivateKey string
	ServiceID  string
	TemplateID string
	httpClient *http.Client
	configured bool
}

func NewClient(publicKey, privateKey, serviceID, templateID string) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	if publicKey != "" && privateKey != "" && serviceID != "" && templateID != "" {
		c.PublicKey = publicKey
		c.PrivateKey = privateKey
		c.ServiceID = serviceID
		c.TemplateID = templateID
		c.configured = true
	}

	return c
}

func (c *Client) IsConfigured() bool {
	return c.configured
}

type emailRequest struct {
	ServiceID      string            `json:"service_id"`
	TemplateID     string            `json:"template_id"`
	UserID         string            `json:"user_id"`
	AccessToken    string            `json:"accessToken"`
	TemplateParams map[string]string `json:"template_params"`
}

func (c *Client) SendEmail(ctx context.Context, toEmail, otp string) error {
	if !c.configured {
		return fmt.Errorf("emailjs client not configured")
	}

	body := emailRequest{
		ServiceID:  c.ServiceID,
		TemplateID: c.TemplateID,
		UserID:     c.PublicKey,
		// AccessToken: c.PrivateKey,
		AccessToken: c.PrivateKey,

		TemplateParams: map[string]string{
			"user":       "username",
			"app_name":   "ChatApp",
			"user_email": toEmail,
			"otp":        otp,
			"time":       "15 minutes",
			"email":      toEmail,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, emailJSAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("EmailJS ERROR %d (Check Template Variables!)", resp.StatusCode)
	}

	return nil
}
