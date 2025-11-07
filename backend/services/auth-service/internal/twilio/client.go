package twilio

import (
	"context"
	"errors"
	"fmt"

	twilioLib "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type Client struct {
	api        *twilioLib.RestClient
	from       string
	configured bool
}

func NewClient(accountSID, authToken, from string) *Client {
	c := &Client{}
	if accountSID != "" && authToken != "" && from != "" {
		c.api = twilioLib.NewRestClientWithParams(twilioLib.ClientParams{
			Username: accountSID,
			Password: authToken,
		})
		c.from = from
		c.configured = true
	} else {
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

func (c *Client) IsConfigured() bool {
	return c.configured
}

func (c *Client) SendSMS(ctx context.Context, to, body string) error {
	if !c.configured {
		return errors.New("twilio client not configured, SMS skipped")
	}

	if to == "" || body == "" {
		return errors.New("recipient 'to' number and 'body' cannot be empty")
	}

	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(c.from)
	params.SetBody(body)

	_, err := c.api.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("twilio send SMS failed for %s: %w", to, err)
	}
	return nil
}
