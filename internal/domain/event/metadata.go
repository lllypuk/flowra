package event

import "time"

// EventMetadata содержит метаданные события
type EventMetadata struct {
	UserID        string
	CorrelationID string
	CausationID   string
	Timestamp     time.Time
	IPAddress     string
	UserAgent     string
}

// NewMetadata создает новые метаданные
func NewMetadata(userID, correlationID, causationID string) EventMetadata {
	return EventMetadata{
		UserID:        userID,
		CorrelationID: correlationID,
		CausationID:   causationID,
		Timestamp:     time.Now(),
	}
}

// WithIPAddress добавляет IP адрес
func (m EventMetadata) WithIPAddress(ip string) EventMetadata {
	m.IPAddress = ip
	return m
}

// WithUserAgent добавляет User-Agent
func (m EventMetadata) WithUserAgent(ua string) EventMetadata {
	m.UserAgent = ua
	return m
}
