package model

import (
	"database/sql"
	"log/slog"
	"time"
)

// SessionStore provides SQLite-backed session storage for scs.
type SessionStore struct {
	db         *sql.DB
	stopCleanup chan struct{}
}

// NewSessionStore creates a new SessionStore and starts periodic cleanup.
func NewSessionStore(db *sql.DB) *SessionStore {
	s := &SessionStore{
		db:         db,
		stopCleanup: make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// StopCleanup stops the periodic cleanup goroutine.
func (s *SessionStore) StopCleanup() {
	close(s.stopCleanup)
}

func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.deleteExpired(); err != nil {
				slog.Error("session cleanup error", "error", err)
			}
		case <-s.stopCleanup:
			return
		}
	}
}

func (s *SessionStore) deleteExpired() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expiry < ?", time.Now())
	return err
}

// Find implements scs.Store.Find
func (s *SessionStore) Find(token string) (b []byte, found bool, err error) {
	row := s.db.QueryRow("SELECT data FROM sessions WHERE token = ? AND expiry > ?", token, time.Now())
	var data []byte
	if err := row.Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

// Commit implements scs.Store.Commit
func (s *SessionStore) Commit(token string, b []byte, expiry time.Time) error {
	_, err := s.db.Exec(
		"INSERT INTO sessions (token, data, expiry) VALUES (?, ?, ?) ON CONFLICT(token) DO UPDATE SET data = ?, expiry = ?",
		token, b, expiry, b, expiry,
	)
	return err
}

// Delete implements scs.Store.Delete
func (s *SessionStore) Delete(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// All implements scs.Store.All (returns ALL session data, used by scs internally for some operations)
func (s *SessionStore) All() (map[string][]byte, error) {
	rows, err := s.db.Query("SELECT token, data FROM sessions WHERE expiry > ?", time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]byte)
	for rows.Next() {
		var token string
		var data []byte
		if err := rows.Scan(&token, &data); err != nil {
			return nil, err
		}
		result[token] = data
	}
	return result, rows.Err()
}
