package notifier

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SMSNotifier sends SMS via Twilio REST API
type SMSNotifier struct {
	AccountSID string
	AuthToken  string
	From       string
	client     *http.Client
	logger     *zap.SugaredLogger
}

func NewSMSNotifier(sid, token, from string, logger *zap.SugaredLogger) *SMSNotifier {
	return &SMSNotifier{
		AccountSID: sid,
		AuthToken: token,
		From: from,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

func (s *SMSNotifier) SendSMS(ctx context.Context, to, message string) error {
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.AccountSID)
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", s.From)
	data.Set("Body", message)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil { return err }
	req.SetBasicAuth(s.AccountSID, s.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.logger.Infof("sms sent to %s", to)
		return nil
	}
	s.logger.Warnf("twilio send failed status=%d", resp.StatusCode)
	return fmt.Errorf("twilio send failed status=%d", resp.StatusCode)
}
