package service

import (
	"database/sql"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/models"
	"github.com/google/uuid"
)

func CreateSession(db *sql.DB, name, hostID string) (*models.Session, error) {
	session := &models.Session{
		ID:        uuid.New().String(),
		Name:      name,
		HostID:    hostID,
		CreatedAt: time.Now(),
	}

	_, err := db.Exec(
		"INSERT INTO sessions VALUES (?, ?, ?, ?)",
		session.ID, session.Name, session.HostID, session.CreatedAt,
	)
	return session, err
}

func ListSessions(db *sql.DB) ([]models.Session, error) {
	rows, err := db.Query("SELECT id, name, host_id, created_at FROM sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		rows.Scan(&s.ID, &s.Name, &s.HostID, &s.CreatedAt)
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func DeleteSession(db *sql.DB, sessionID, hostID string) error {
	// Verify the session belongs to this host before deleting
	var existingHostID string
	err := db.QueryRow("SELECT host_id FROM sessions WHERE id = ?", sessionID).Scan(&existingHostID)
	if err != nil {
		return err
	}
	
	if existingHostID != hostID {
		return sql.ErrNoRows // Not authorized
	}
	
	_, err = db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}
