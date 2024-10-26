package auth

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"social-network/backend/model"
)

func AuthMiddleware(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("AuthMiddleware invoked")

		// Clean up expired sessions
		_, cleanupErr := db.Exec("DELETE FROM Sessions WHERE ExpiresAt < ?", time.Now())
		if cleanupErr != nil {
			log.Printf("Error cleaning up sessions: %v", cleanupErr)
		}

		log.Println("Clean-up query executed")

		// Retrieve session_id cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		sessionID := cookie.Value

		// Validate the session
		isValid, err := model.ValidateSession(db, sessionID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !isValid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("Trying to extend session %s", sessionID)

		// Extend session expiry
		err = model.ExtendSessionExpiry(db, sessionID)
		if err != nil {
			log.Printf("Error extending session: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Update the session cookie expiration time
		expirationTime := time.Now().Add(45 * time.Minute)
		cookie.Expires = expirationTime
		http.SetCookie(w, cookie)

		log.Printf("Session %s and cookie extended", sessionID)

		// Call the next handler
		next.ServeHTTP(w, r)
	}
}

func EnableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:8081")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
}
