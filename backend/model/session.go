package model

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"time"
)

// GenerateSessionID generates a random session ID
func GenerateSessionID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		log.Printf("Error generating session ID: %v", err)
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession inserts a new session into the datab or updates the existing one
func CreateSession(db *sql.DB, sessionID string, userID int, expiration time.Time) error {
	// Check if a session already exists for the user
	var existingSessionID string
	err := db.QueryRow("SELECT SessionID FROM Sessions WHERE UserID = ?", userID).Scan(&existingSessionID)

	if err == sql.ErrNoRows {
		// No existing session, create a new one
		_, err = db.Exec("INSERT INTO Sessions (UserID, SessionID, ExpiresAt) VALUES (?, ?, ?)", userID, sessionID, expiration)
		if err != nil {
			log.Printf("Error creating new session: %v", err)
			return err
		}
		log.Println("New session created successfully")
	} else if err == nil {
		// Existing session found, update it
		_, err = db.Exec("UPDATE Sessions SET SessionID = ?, ExpiresAt = ? WHERE UserID = ?", sessionID, expiration, userID)
		if err != nil {
			log.Printf("Error updating existing session: %v", err)
			return err
		}
		log.Println("Existing session updated successfully")
	} else {
		// Some other error occurred
		log.Printf("Error checking for existing session: %v", err)
		return err
	}

	return nil
}

// ValidateSession checks if a session is valid and not expired
func ValidateSession(db *sql.DB, sessionID string) (bool, error) {
	var expiresAt time.Time

	err := db.QueryRow("SELECT ExpiresAt FROM Sessions WHERE SessionID = ?", sessionID).Scan(&expiresAt)
	log.Printf("Validating session with ID: %s, expires at: %v", sessionID, expiresAt) // Moved after DB read

	if err != nil {
		return false, err
	}

	if time.Now().After(expiresAt) {
		return false, nil // Session has expired
	}
	return true, nil // Session is valid
}

// ExtendSessionExpiry updates the expiry time of a session
func ExtendSessionExpiry(db *sql.DB, sessionID string) error {
	var expiresAt time.Time

	// Reading old expiry time for logging
	err := db.QueryRow("SELECT ExpiresAt FROM Sessions WHERE SessionID = ?", sessionID).Scan(&expiresAt)
	if err != nil {
		return err
	}

	newExpiresAt := time.Now().Add(45 * time.Minute)
	log.Printf("Old expiry time: %v, New expiry time: %v", expiresAt, newExpiresAt)

	_, err = db.Exec("UPDATE Sessions SET ExpiresAt = ? WHERE SessionID = ?", newExpiresAt, sessionID)
	return err
}

// DeleteSession removes a session from the datab
func DeleteSession(db *sql.DB, sessionID string) error {
	_, err := db.Exec("DELETE FROM Sessions WHERE SessionID = ?", sessionID)
	return err
}

// CleanExpiredSessions periodically cleans up expired sessions in the main.go with a go routine
func CleanExpiredSessions(db *sql.DB) {
	// Immediately perform cleanup before starting the ticker
	cleanupExpiredSessions(db)

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cleanupExpiredSessions(db)
		}
	}
}

func cleanupExpiredSessions(db *sql.DB) {
	log.Println("Checking for expired sessions...")

	rows, err := db.Query("SELECT SessionID FROM Sessions WHERE ExpiresAt < ?", time.Now())
	if err != nil {
		log.Printf("Error fetching expired sessions: %v", err)
		return
	}
	defer rows.Close()

	var sessionIDs []string
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			log.Printf("Error scanning session ID: %v", err)
			continue
		}
		sessionIDs = append(sessionIDs, sessionID)
	}

	if len(sessionIDs) == 0 {
		log.Println("No expired sessions found.")
	} else {
		log.Printf("Found %d expired sessions. Deleting...", len(sessionIDs))
	}

	for _, id := range sessionIDs {
		if err := DeleteSession(db, id); err != nil {
			log.Printf("Error deleting session %s: %v", id, err)
		} else {
			log.Printf("Successfully deleted expired session: %s", id)
		}
	}
}

func GetUserIDBySessionID(db *sql.DB, sessionID string) (int, error) {
	var userID int
	row := db.QueryRow("SELECT UserID FROM Sessions WHERE SessionID = ?", sessionID)
	err := row.Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}
