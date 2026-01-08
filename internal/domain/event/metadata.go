package event

import "time"

// Metadata contains event metadata
type Metadata struct {
	UserID        string    `json:"user_id,omitempty"        bson:"user_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty" bson:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"   bson:"causation_id,omitempty"`
	Timestamp     time.Time `json:"timestamp,omitempty"      bson:"timestamp,omitempty"`
	IPAddress     string    `json:"ip_address,omitempty"     bson:"ip_address,omitempty"`
	UserAgent     string    `json:"user_agent,omitempty"     bson:"user_agent,omitempty"`
}

// NewMetadata creates new metadata
func NewMetadata(userID, correlationID, causationID string) Metadata {
	return Metadata{
		UserID:        userID,
		CorrelationID: correlationID,
		CausationID:   causationID,
		Timestamp:     time.Now(),
	}
}

// WithIPAddress adds IP address
func (m Metadata) WithIPAddress(ip string) Metadata {
	m.IPAddress = ip
	return m
}

// WithUserAgent adds User-Agent
func (m Metadata) WithUserAgent(ua string) Metadata {
	m.UserAgent = ua
	return m
}
