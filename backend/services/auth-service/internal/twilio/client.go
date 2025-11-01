package twilio

import (
	"context"
	"fmt"
	"io" // Added import for io
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client defines the Twilio SMS client interface
type Client interface {
	SendSMS(ctx context.Context, toPhoneNumber, message string) error
}

// twilioClient implements the Client interface
type twilioClient struct {
	accountSID string
	authToken  string
	fromNumber string
	httpClient *http.Client
}

// NewClient creates a new Twilio client
func NewClient(accountSID, authToken, fromNumber string) Client {
	return &twilioClient{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		httpClient: &http.Client{Timeout: 10 * time.Second}, // Add a timeout to the HTTP client
	}
}

// SendSMS sends an SMS message via Twilio
func (tc *twilioClient) SendSMS(ctx context.Context, toPhoneNumber, message string) error {
	twilioURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", tc.accountSID)

	data := url.Values{}
	data.Set("To", toPhoneNumber)
	data.Set("From", tc.fromNumber)
	data.Set("Body", message)
	encodedData := data.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", twilioURL, strings.NewReader(encodedData))
	if err != nil {
		return fmt.Errorf("failed to create Twilio SMS request: %w", err)
	}

	req.SetBasicAuth(tc.accountSID, tc.authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Twilio SMS request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Read the response body for more detailed error from Twilio
		var errorBody string
		buf := new(strings.Builder)
		_, _ = io.Copy(buf, resp.Body) // Changed from buf.ReadFrom to io.Copy
		errorBody = buf.String()
		return fmt.Errorf("Twilio API returned non-success status: %d - %s", resp.StatusCode, errorBody)
	}

	// You could parse the response body here to get the SID or other details if needed
	return nil
}
