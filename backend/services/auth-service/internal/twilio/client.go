package twilio

import (
	"context"
	"errors"
	"fmt"

	twilioLib "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Client represents the Twilio API client.
type Client struct {
	api        *twilioLib.RestClient
	from       string
	configured bool // Flag to indicate if the client is properly configured
}

// NewClient creates and returns a new Twilio Client.
// It checks if accountSID, authToken, and from number are provided to mark it as configured.
func NewClient(accountSID, authToken, from string) *Client {
	c := &Client{}
	// Check if all necessary credentials are provided
	if accountSID != "" && authToken != "" && from != "" {
		// Twilio client can also read credentials from environment variables TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN
		// but explicit passing is clearer.
		// If using env vars directly, ensure they are set when NewRestClient is called without params.
		c.api = twilioLib.NewRestClientWithParams(twilioLib.ClientParams{
			Username: accountSID,
			Password: authToken,
		})
		c.from = from
		c.configured = true // Mark as configured
	} else {
		// Log which parameters are missing for easier debugging
		if accountSID == "" {
			fmt.Println("[twilio] Warning: TWILIO_ACCOUNT_SID is empty.")
		}
		if authToken == "" {
			fmt.Println("[twilio] Warning: TWILIO_AUTH_TOKEN is empty.")
		}
		if from == "" {
			fmt.Println("[twilio] Warning: TWILIO_FROM is empty.")
		}
	}
	return c
}

// IsConfigured returns true if the Twilio client has been initialized with the necessary credentials.
func (c *Client) IsConfigured() bool {
	return c.configured
}

// SendSMS sends an SMS message using the Twilio API.
// It will be a no-op if the client is not configured, returning nil.
func (c *Client) SendSMS(ctx context.Context, to, body string) error {
	if !c.configured {
		// Log this instead of just printing, as it's a condition where an SMS *might* have been expected.
		return errors.New("twilio client not configured, SMS skipped")
	}

	if to == "" || body == "" {
		return errors.New("recipient 'to' number and 'body' cannot be empty")
	}

	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(c.from)
	params.SetBody(body)

	// CreateMessage automatically uses the context passed to the client or the context from the call.
	// The twilio-go library internally handles the HTTP request with the context.
	_, err := c.api.Api.CreateMessage(params)
	if err != nil {
		// Twilio errors can be quite verbose, consider wrapping for cleaner error messages
		// For example, check for specific error codes if needed.
		return fmt.Errorf("twilio send SMS failed for %s: %w", to, err)
	}
	return nil
}
