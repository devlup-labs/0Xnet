package identity

import "github.com/google/uuid"

func NewDeviceID() string {
	return uuid.New().String()
}
