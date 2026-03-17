package service

import (
	"database/sql"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/models"
	"github.com/google/uuid"
)

// JoinSession adds a device as a member of a session (idempotent — won't duplicate)
func JoinSession(db *sql.DB, sessionID, deviceID, deviceName string) (*models.SessionMember, error) {
	// Check if already a member
	var existingID string
	err := db.QueryRow(
		"SELECT id FROM session_members WHERE session_id = ? AND device_id = ?",
		sessionID, deviceID,
	).Scan(&existingID)

	if err == nil {
		// Already a member, return existing
		return getSessionMemberByID(db, existingID)
	}

	// Verify the session exists
	var sessID string
	err = db.QueryRow("SELECT id FROM sessions WHERE id = ?", sessionID).Scan(&sessID)
	if err != nil {
		return nil, err
	}

	member := &models.SessionMember{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		DeviceID:   deviceID,
		DeviceName: deviceName,
		JoinedAt:   time.Now(),
	}

	_, err = db.Exec(
		"INSERT INTO session_members (id, session_id, device_id, device_name, joined_at) VALUES (?, ?, ?, ?, ?)",
		member.ID, member.SessionID, member.DeviceID, member.DeviceName, member.JoinedAt,
	)
	if err != nil {
		return nil, err
	}

	return member, nil
}

// LeaveSession removes a device from a session
func LeaveSession(db *sql.DB, sessionID, deviceID string) error {
	result, err := db.Exec(
		"DELETE FROM session_members WHERE session_id = ? AND device_id = ?",
		sessionID, deviceID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetSessionMembers returns all members of a session
func GetSessionMembers(db *sql.DB, sessionID string) ([]models.SessionMember, error) {
	rows, err := db.Query(
		"SELECT id, session_id, device_id, device_name, joined_at FROM session_members WHERE session_id = ?",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.SessionMember
	for rows.Next() {
		var m models.SessionMember
		if err := rows.Scan(&m.ID, &m.SessionID, &m.DeviceID, &m.DeviceName, &m.JoinedAt); err != nil {
			continue
		}
		members = append(members, m)
	}

	if members == nil {
		members = []models.SessionMember{}
	}
	return members, nil
}

// IsSessionMember checks if a device is already a member of a session
func IsSessionMember(db *sql.DB, sessionID, deviceID string) bool {
	var id string
	err := db.QueryRow(
		"SELECT id FROM session_members WHERE session_id = ? AND device_id = ?",
		sessionID, deviceID,
	).Scan(&id)
	return err == nil
}

// DeleteSessionMembers removes all members from a session (used when session is deleted)
func DeleteSessionMembers(db *sql.DB, sessionID string) error {
	_, err := db.Exec("DELETE FROM session_members WHERE session_id = ?", sessionID)
	return err
}

// getSessionMemberByID fetches a single member by ID
func getSessionMemberByID(db *sql.DB, id string) (*models.SessionMember, error) {
	var m models.SessionMember
	err := db.QueryRow(
		"SELECT id, session_id, device_id, device_name, joined_at FROM session_members WHERE id = ?",
		id,
	).Scan(&m.ID, &m.SessionID, &m.DeviceID, &m.DeviceName, &m.JoinedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
