package twilio

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "time"
)

type Client struct {
    AccountSID string
    AuthToken  string
    From       string
    httpClient *http.Client
}

func NewClient(sid, token, from string) *Client {
    return &Client{
        AccountSID: sid,
        AuthToken:  token,
        From:       from,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *Client) SendSMS(ctx context.Context, to string, body string) error {
    // POST to https://api.twilio.com/2010-04-01/Accounts/{AccountSid}/Messages.json
    endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.AccountSID)
    data := url.Values{}
    data.Set("To", to)
    data.Set("From", c.From)
    data.Set("Body", body)

    req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
    if err != nil {
        return err
    }
    req.SetBasicAuth(c.AccountSID, c.AuthToken)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return nil
    }
    var respBody map[string]any
    _ = json.NewDecoder(resp.Body).Decode(&respBody)
    return fmt.Errorf("twilio send failed status=%d body=%v", resp.StatusCode, respBody)
}
