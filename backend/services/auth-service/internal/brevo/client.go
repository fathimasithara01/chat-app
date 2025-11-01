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
    APIKey     string
    FromEmail  string
    FromName   string
    httpClient *http.Client
}

func NewClient(apiKey, fromEmail, fromName string) *Client {
    return &Client{
        APIKey:     apiKey,
        FromEmail:  fromEmail,
        FromName:   fromName,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

type sendEmailReq struct {
    Sender struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    } `json:"sender"`
    To []map[string]string `json:"to"`
    Subject string `json:"subject"`
    HTMLContent string `json:"htmlContent,omitempty"`
    TextContent string `json:"textContent,omitempty"`
}

func (c *Client) SendEmail(ctx context.Context, toEmail, subject, htmlContent, textContent string) error {
    reqBody := sendEmailReq{}
    reqBody.Sender.Name = c.FromName
    reqBody.Sender.Email = c.FromEmail
    reqBody.To = []map[string]string{{"email": toEmail}}
    reqBody.Subject = subject
    reqBody.HTMLContent = htmlContent
    reqBody.TextContent = textContent

    buf := &bytes.Buffer{}
    if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", "https://api.brevo.com/v3/smtp/email", buf)
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("api-key", c.APIKey)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return nil
    }
    var r map[string]any
    _ = json.NewDecoder(resp.Body).Decode(&r)
    return fmt.Errorf("brevo send failed status=%d body=%v", resp.StatusCode, r)
}
