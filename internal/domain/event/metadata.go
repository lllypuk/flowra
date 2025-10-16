package event

import "time"

// Metadata содержит метаданные события
type Metadata struct {
	UserID        string
	CorrelationID string
	CausationID   string
	Timestamp     time.Time
	IPAddress     string
	UserAgent     string
}

// NewMetadata создает новые метаданные
func NewMetadata(userID, correlationID, causationID string) Metadata {
	return Metadata{
		UserID:        userID,
		CorrelationID: correlationID,
		CausationID:   causationID,
		Timestamp:     time.Now(),
	}
}

// WithIPAddress добавляет IP адрес
func (m Metadata) WithIPAddress(ip string) Metadata {
	m.IPAddress = ip
	return m
}

// WithUserAgent добавляет User-Agent
func (m Metadata) WithUserAgent(ua string) Metadata {
	m.UserAgent = ua
	return m
}
