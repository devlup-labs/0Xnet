package models

import "time"

type Session struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	HostID    string          `json:"hostId"`
	CreatedAt time.Time       `json:"createdAt"`
	HostIP    string          `json:"hostIp,omitempty"`
	HostPort  int             `json:"hostPort,omitempty"`
	Members   []SessionMember `json:"members,omitempty"`
}
