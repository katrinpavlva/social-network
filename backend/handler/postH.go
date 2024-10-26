package handler

import (
	"cloud.google.com/go/storage"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"

	"social-network/backend/auth"
	"social-network/backend/datab"
	"social-network/backend/model"
)

func CreatePH(db *sql.DB, storageClient *storage.Client, bucketName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		cookie, err := r.Cookie("session_id")
		if err != nil {
			log.Printf("Error retrieving session: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		sessionID := cookie.Value

		// Retrieve the user_id based on the session_id
		userID, err := model.GetUserIDBySessionID(db, sessionID)
		if err != nil {
			log.Printf("Error retrieving user ID: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Process the image only if it's provided
		var imageURL string
		file, header, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			newFileName := "posts/" + uuid.New().String() + "_" + header.Filename
			imageURL, err = datab.StoreToCloud(context.Background(), storageClient, bucketName, newFileName, file)
			if err != nil {
				log.Printf("Failed to upload image: %v", err)
				http.Error(w, "Failed to upload image", http.StatusInternalServerError)
				return
			}
		} else if err != http.ErrMissingFile {
			// Handle other errors
			http.Error(w, "Error processing image file", http.StatusBadRequest)
			return
		}

		allowedViewers := r.FormValue("selectedUserIds")
		if allowedViewers == "" {
			allowedViewers = "[]" // Use an empty JSON array to represent no viewers
		}

		log.Printf("Allowed Viewers: %v", allowedViewers)

		groupIDParam := r.FormValue("groupID")
		var groupID sql.NullInt64
		if groupIDParam != "" {
			groupIDInt, err := strconv.Atoi(groupIDParam) // Convert string to int
			if err != nil {
				log.Printf("Error converting groupID to int: %v", err)
				http.Error(w, "Invalid groupID", http.StatusBadRequest)
				return
			}
			groupID = sql.NullInt64{Int64: int64(groupIDInt), Valid: true}
		} else {
			groupID = sql.NullInt64{Valid: false} // GroupID is null
		}

		newPost := model.Post{
			UserID:         userID,
			Content:        r.FormValue("content"),
			PrivacySetting: r.FormValue("privacy"),
			ImageURL:       imageURL,
			AllowedViewers: allowedViewers,
			GroupID:        groupID,
		}

		createdPost, err := model.CreatePost(db, newPost)
		if err != nil {
			log.Printf("Error creating post: %v", err)
			http.Error(w, "Error creating post", http.StatusInternalServerError)
			return
		}

		// Respond with the newly created post
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(createdPost); err != nil {
			log.Printf("Error sending post response: %v", err)
			http.Error(w, "Error sending post response", http.StatusInternalServerError)
			return
		}
	}
}

func GetPH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w) // Make sure to adjust EnableCors to accept *http.Request if needed
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		groupID := r.URL.Query().Get("groupID")
		var posts []model.Post
		var err error

		if groupID != "" {
			// Fetch posts for a specific group
			posts, err = model.GetGroupPosts(db, groupID)
		} else {
			// Fetch posts that don't belong to any group
			posts, err = model.GetPosts(db)
		}

		if err != nil {
			log.Printf("Error fetching posts: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	}
}
