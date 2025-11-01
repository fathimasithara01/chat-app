package event_handler

import (
	"context"
	"encoding/json"
	"errors"
	"notification-service/internal/notifier"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Event model expected on Kafka - extend as needed
type Event struct {
	Type       string         `json:"type"`     // "otp","welcome","message"
	Receiver   string         `json:"receiver"` // email or phone
	Payload    map[string]any `json:"payload"`
	Template   string         `json:"template,omitempty"` // template key for emails
	RetryCount int            `json:"retry_count,omitempty"`
}

type Handler struct {
	email          *notifier.EmailNotifier
	sms            *notifier.SMSNotifier
	logger         *zap.SugaredLogger
	dlqWriter      *kafka.Writer
	maxRetries     int
	retryBackoffMs int
}

func NewHandler(email *notifier.EmailNotifier, sms *notifier.SMSNotifier, dlqWriter *kafka.Writer, maxRetries, backoffMs int, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		email: email, sms: sms, dlqWriter: dlqWriter,
		maxRetries: maxRetries, retryBackoffMs: backoffMs,
		logger: logger,
	}
}

// HandleEvent executes the correct notifier depending on event type.
// It returns nil on success. On repeated failure it will push message to DLQ.
func (h *Handler) HandleEvent(ctx context.Context, raw []byte) error {
	var ev Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		h.logger.Errorf("invalid event: %v", err)
		return err
	}
	// attempt with retry loop (simple exponential backoff)
	var lastErr error
	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		if attempt > 0 {
			sleep := time.Duration(h.retryBackoffMs*(1<<uint(attempt-1))) * time.Millisecond
			h.logger.Infof("retry attempt %d sleeping %v", attempt, sleep)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleep):
			}
		}
		if err := h.processOnce(ctx, &ev); err != nil {
			lastErr = err
			h.logger.Warnf("process attempt failed: %v", err)
			continue
		}
		// success
		return nil
	}
	// all retries failed -> push to DLQ
	h.logger.Errorf("pushing to DLQ after %d attempts: lastErr=%v", h.maxRetries, lastErr)
	if err := h.pushToDLQ(ctx, raw); err != nil {
		h.logger.Errorf("dlq push failed: %v", err)
		// if DLQ also fails, return lastErr so caller may handle
		return errors.New("notify failed and dlq push failed")
	}
	return lastErr
}

func (h *Handler) processOnce(ctx context.Context, ev *Event) error {
	switch ev.Type {
	case "otp":
		// payload.message required and receiver is phone
		msg, _ := ev.Payload["message"].(string)
		if msg == "" {
			return errors.New("otp event missing message")
		}
		return h.sms.SendSMS(ctx, ev.Receiver, msg)

	case "welcome":
		// email welcome - template in ev.Template or default
		subject := ev.Payload["subject"].(string)
		if subject == "" {
			subject = "Welcome!"
		}
		tpl := ev.Template
		if tpl == "" {
			tpl = "welcome"
		}
		return h.email.SendTemplateEmail(ctx, ev.Receiver, subject, tpl, ev.Payload)

	case "message":
		// push email for offline user
		subject := ev.Payload["subject"].(string)
		if subject == "" {
			subject = "New message"
		}
		tpl := ev.Template
		if tpl == "" {
			tpl = "chat_message"
		}
		return h.email.SendTemplateEmail(ctx, ev.Receiver, subject, tpl, ev.Payload)

	default:
		return errors.New("unknown event type")
	}
}

func (h *Handler) pushToDLQ(ctx context.Context, raw []byte) error {
	msg := kafka.Message{
		Key:   nil,
		Value: raw,
		Time:  time.Now(),
	}
	return h.dlqWriter.WriteMessages(ctx, msg)
}
