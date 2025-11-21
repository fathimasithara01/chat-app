package ws

import "encoding/json"

// Envelope is the standard wire format for ws messages
type Envelope struct {
	Type    string          `json:"type"`
	ChatID  string          `json:"chat_id,omitempty"`
	MsgID   string          `json:"msg_id,omitempty"`
	From    string          `json:"from,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}
