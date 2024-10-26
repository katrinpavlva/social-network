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

func CrComHandler(db *sql.DB, storageClient *storage.Client, bucketName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("-------------- Inside CrComHandler ------------------")

		// Parse the multipart form
		err := r.ParseMultipartForm(32 << 20) // maxMemory 32MB
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Extract the text fields
		postID := r.FormValue("postID")
		userID := r.FormValue("userID")
		content := r.FormValue("content")

		// Convert postID and userID to integers
		postIDInt, err := strconv.Atoi(postID)
		if err != nil {
			log.Printf("Invalid postID: %v", err)
			http.Error(w, "Invalid postID", http.StatusBadRequest)
			return
		}
		userIDInt, err := strconv.Atoi(userID)
		if err != nil {
			log.Printf("Invalid userID: %v", err)
			http.Error(w, "Invalid userID", http.StatusBadRequest)
			return
		}

		// Assign the values to newComment
		newComment := model.Comment{
			PostID:  postIDInt,
			UserID:  userIDInt,
			Content: content,
		}

		// Process the image file if present
		file, header, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			newFileName := "comments/" + uuid.New().String() + "_" + header.Filename
			imageURL, err := datab.StoreToCloud(context.Background(), storageClient, bucketName, newFileName, file)
			if err != nil {
				log.Printf("Failed to upload image: %v", err)
				http.Error(w, "Failed to upload image", http.StatusInternalServerError)
				return
			}
			newComment.CommentMedia = imageURL
		} else if err != http.ErrMissingFile {
			log.Printf("Error processing image file: %v", err)
			http.Error(w, "Error processing image file", http.StatusBadRequest)
			return
		}

		// Insert the new comment into the datab
		createdComment, err := model.CreateComment(db, newComment)
		if err != nil {
			log.Printf("Error creating comment: %v", err)
			http.Error(w, "Error creating comment", http.StatusInternalServerError)
			return
		}

		log.Println("Comment created successfully, returning new comment")

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(createdComment); err != nil {
			log.Printf("Error sending comment response: %v", err)
			http.Error(w, "Error sending comment response", http.StatusInternalServerError)
			return
		}
	}
}

func GePostComH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("--------------- Inside GePostComH ---------------")

		postID := r.URL.Query().Get("postID")
		if postID == "" {
			http.Error(w, "postID is required", http.StatusBadRequest)
			return
		}

		// Call the GetCommentsForPost function which executes the datab query
		comments, err := model.GetCommentsForPost(db, postID)
		if err != nil {
			log.Printf("Error fetching comments: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Println("Returning comments")
		// Set the header and write the response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comments)
	}
}
