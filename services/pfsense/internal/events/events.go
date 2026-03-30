package events

import "time"

type Envelope struct {
	Type      string         `json:"type"`
	Source    string         `json:"source"`
	Timestamp string         `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

type Publisher interface {
	Publish(topic string, event Envelope) error
}

func NewEnvelope(eventType, source string, payload map[string]any) Envelope {
	return Envelope{
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	}
}
