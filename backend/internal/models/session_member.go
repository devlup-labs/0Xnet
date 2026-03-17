package models

import "time"

// SessionMember represents a device that has joined a session
type SessionMember struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"sessionId"`
	DeviceID   string    `json:"deviceId"`
	DeviceName string    `json:"deviceName"`
	JoinedAt   time.Time `json:"joinedAt"`
}
