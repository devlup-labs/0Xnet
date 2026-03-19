package service

import (
	"database/sql"
	"log"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/models"
	"github.com/google/uuid"
)

// CleanupStaleSessions removes all sessions (and their members) that don't
// belong to the current deviceID. This handles the case where the server
// restarted with a new deviceID but old sessions are still in the DB.
func CleanupStaleSessions(db *sql.DB, currentDeviceID string) {
	rows, err := db.Query("SELECT id FROM sessions WHERE host_id != ?", currentDeviceID)
	if err != nil {
		return
	}
	defer rows.Close()

	var staleIDs []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			staleIDs = append(staleIDs, id)
		}
	}

	for _, id := range staleIDs {
		_ = DeleteSessionMembers(db, id)
		db.Exec("DELETE FROM sessions WHERE id = ?", id)
	}

	if len(staleIDs) > 0 {
		log.Printf("🧹 Cleaned up %d stale session(s) from previous runs", len(staleIDs))
	}
}

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
	if err != nil {
		return nil, err
	}

	// Auto-add the host as the first member of the session
	_, _ = JoinSession(db, session.ID, hostID, "Host")

	return session, nil
}

func ListSessions(db *sql.DB, hostID string) ([]models.Session, error) {
	rows, err := db.Query("SELECT id, name, host_id, created_at FROM sessions WHERE host_id = ? ORDER BY created_at DESC", hostID)
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

	// Cascade: delete all members of this session first
	_ = DeleteSessionMembers(db, sessionID)

	_, err = db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}
