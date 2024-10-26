package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"social-network/backend/auth"
	"social-network/backend/model"
	"strconv"
)

func GetUserPH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("-------------- Inside GetUserPH ------------------")

		userIdStr := r.URL.Query().Get("userId")
		if userIdStr == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		userId, err := strconv.Atoi(userIdStr)
		if err != nil {
			log.Printf("Error converting userId to int: %v", err)
			http.Error(w, "Invalid User ID", http.StatusBadRequest)
			return
		}

		posts, err := model.FetchPostsByUserID(db, userId)
		if err != nil {
			log.Printf("Error fetching posts for user %d: %v", userId, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(posts); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func GetFollowH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("-------------- Inside GetUserPH ------------------")

		userIdStr := r.URL.Query().Get("userId")
		if userIdStr == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		userId, err := strconv.Atoi(userIdStr)
		if err != nil {
			log.Printf("Error converting userId to int: %v", err)
			http.Error(w, "Invalid User ID", http.StatusBadRequest)
			return
		}

		following, errFollowing := model.FetchFollowingByUserID(db, userId)
		if errFollowing != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Fetch followers
		followers, errFollowers := model.FetchFollowersByUserID(db, userId)
		if errFollowers != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Combine following and followers
		combinedUsers := append(following, followers...)

		// Send combined response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(combinedUsers)
	}
}

func GetUserDetH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		userIDStr := r.URL.Query().Get("userId")
		if userIDStr == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w, "Invalid User ID", http.StatusBadRequest)
			return
		}

		var user struct {
			UserID         int    `json:"userID"`
			Email          string `json:"email"`
			FirstName      string `json:"firstName"`
			LastName       string `json:"lastName"`
			DateOfBirth    string `json:"dateOfBirth"`
			ProfilePicture string `json:"profilePicture"`
			Nickname       string `json:"nickname"`
			AboutMe        string `json:"aboutMe"`
			Gender         string `json:"gender"`
			CreatedAt      string `json:"createdAt"`
			ProfilePrivacy string `json:"profilePrivacy"`
		}

		query := `SELECT UserID, Email, FirstName, LastName, DateOfBirth, ProfilePicture, Nickname, AboutMe, Gender, CreatedAt, ProfilePrivacy FROM User WHERE UserID = ?`
		err = db.QueryRow(query, userID).Scan(&user.UserID, &user.Email, &user.FirstName, &user.LastName, &user.DateOfBirth, &user.ProfilePicture, &user.Nickname, &user.AboutMe, &user.Gender, &user.CreatedAt, &user.ProfilePrivacy)
		if err != nil {
			log.Printf("Error fetching user details: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func ToggleProPrivH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserID         int    `json:"userId"`
			ProfilePrivacy string `json:"profilePrivacy"` // Expected to be either "Private" or "Public"
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Ensure profilePrivacy is either "Private" or "Public"
		if req.ProfilePrivacy != "Private" && req.ProfilePrivacy != "Public" {
			http.Error(w, "Invalid profile privacy setting", http.StatusBadRequest)
			return
		}

		query := `UPDATE User SET ProfilePrivacy = ? WHERE UserID = ?`
		_, err := db.Exec(query, req.ProfilePrivacy, req.UserID)
		if err != nil {
			log.Printf("Error updating profile privacy: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}
