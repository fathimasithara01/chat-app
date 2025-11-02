package twilio

import (
	"context"
	"fmt"

	twilioLib "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type Client struct {
	api  *twilioLib.RestClient
	from string
}

func NewClient(accountSID, authToken, from string) *Client {
	if accountSID == "" || authToken == "" || from == "" {
		return &Client{}
	}
	client := twilioLib.NewRestClientWithParams(twilioLib.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &Client{api: client, from: from}
}

func (c *Client) SendSMS(ctx context.Context, to, body string) error {
	if c.api == nil {
		// noop in dev (not configured)
		return nil
	}
	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(c.from)
	params.SetBody(body)
	_, err := c.api.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("twilio send sms: %w", err)
	}
	return nil
}
