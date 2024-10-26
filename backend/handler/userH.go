package handler

import (
	"cloud.google.com/go/storage"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"time"

	"social-network/backend/auth"
	"social-network/backend/datab"
	"social-network/backend/model"
)

func RegisterH(db *sql.DB, storageClient *storage.Client, bucketName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("---------------- Inside RegisterH ----------------")
		// Check the method of the request
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the multipart form
		err := r.ParseMultipartForm(10 << 20) // Max upload size ~10MB
		if err != nil {
			http.Error(w, "File too large", http.StatusBadRequest)
			return
		}

		// Process the profile picture only if it's provided
		var profilePicURL string
		file, header, err := r.FormFile("profilePicture")
		if err == nil {
			defer file.Close()

			// Generate a unique file name for the profile picture
			newFileName := "profilepics/" + uuid.New().String() + "_" + header.Filename

			// Upload the profile picture to Google Cloud Storage
			profilePicURL, err = datab.StoreToCloud(context.Background(), storageClient, bucketName, newFileName, file)
			if err != nil {
				log.Printf("Failed to upload profile picture: %v", err)
				http.Error(w, fmt.Sprintf("Failed to upload profile picture: %v", err), http.StatusInternalServerError)
				return
			}
		} else if err != http.ErrMissingFile {
			// Handle other errors
			http.Error(w, "Error processing file", http.StatusBadRequest)
			return
		}

		// Manually extract form values for non-omitempty fields
		newUser := model.User{
			Email:          r.FormValue("Email"),
			Password:       r.FormValue("Password"), // This will be hashed before storage
			FirstName:      r.FormValue("FirstName"),
			LastName:       r.FormValue("LastName"),
			Gender:         r.FormValue("Gender"),
			ProfilePicture: profilePicURL,
		}

		// Parse and assign omitempty fields only if they are provided
		dob := r.FormValue("DateOfBirth")
		if dob != "" {
			parsedDOB, err := time.Parse("2006-01-02", dob)
			if err != nil {
				http.Error(w, "Invalid date of birth format", http.StatusBadRequest)
				return
			}
			newUser.DateOfBirth = parsedDOB
		}

		nickname := r.FormValue("Nickname")
		if nickname != "" {
			newUser.Nickname = nickname
		}

		aboutMe := r.FormValue("AboutMe")
		if aboutMe != "" {
			newUser.AboutMe = aboutMe
		}

		profilePrivacy := r.FormValue("ProfilePrivacy")
		if profilePrivacy != "Private" {
			profilePrivacy = "Public" // Default to "Public" if not specified as "Private"
		}
		newUser.ProfilePrivacy = profilePrivacy

		log.Println("Received password for registration:", newUser.Password)
		log.Println("New User:", newUser)

		exists, err := model.UserExists(db, newUser.Email, newUser.Nickname)
		if err != nil {
			// Handle error, maybe log it and return an internal server error
			http.Error(w, `{"error": "Error checking user existence"}`, http.StatusInternalServerError)
			return
		}

		if exists {
			http.Error(w, `{"error": "Email or nickname already in use"}`, http.StatusBadRequest)
			return
		}

		// Hashing the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error": "Invalid login credentials"}`, http.StatusUnauthorized)
			return
		}
		log.Println("Generated hash for registration:", string(hashedPassword))

		newUser.PasswordHash = string(hashedPassword)

		// Inserting the User data into the datab
		err = model.RegisterUser(db, &newUser)
		if err != nil {
			http.Error(w, `{"error": "Specific registration error message"}`, http.StatusBadRequest)
			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		}
	}
}

func LoginH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("---------------- Inside LoginH ----------------")

		// Create struct to match expected JSON
		var creds struct {
			Credential string `json:"credential"`
			Password   string `json:"password"`
		}

		// Parse JSON from request body
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Error parsing JSON", http.StatusBadRequest)
			return
		}

		log.Println("Retrieve credentials:", creds.Credential)
		log.Println("Password:", creds.Password)

		// Attempt to retrieve the user by email or nickname
		user, err := model.GetUserByCredential(db, creds.Credential)
		if err == sql.ErrNoRows {
			// User with provided email or nickname does not exist
			http.Error(w, `{"error": "Email or nickname does not exist"}`, http.StatusUnauthorized)
			return
		} else if err != nil {
			// Handle unexpected error
			http.Error(w, `{"error": "Wrong password"}`, http.StatusInternalServerError)
			return
		}

		// Compare the provided password with the hashed password in the datab
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password))
		if err != nil {
			log.Printf("Password comparison failed for user '%s': %v", user.Email, err)
			http.Error(w, `{"error": "Invalid password"}`, http.StatusInternalServerError)
			return
		}

		// If password is correct, generate a new session ID
		sessionID, err := model.GenerateSessionID()
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		// Set session expiration time
		expiration := time.Now().Add(45 * time.Minute)

		// Create session in the datab
		err = model.CreateSession(db, sessionID, user.UserID, expiration)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		userInfo := map[string]interface{}{
			"userID":         user.UserID,
			"email":          user.Email,
			"firstName":      user.FirstName,
			"lastName":       user.LastName,
			"dateOfBirth":    user.DateOfBirth,
			"profilePicture": user.ProfilePicture,
			"nickname":       user.Nickname,
			"aboutMe":        user.AboutMe,
			"gender":         user.Gender,
			"createdAt":      user.CreatedAt,
			"profilePrivacy": user.ProfilePrivacy,
		}

		// Set the session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Expires:  expiration,
			HttpOnly: false,
			Path:     "/",
			Secure:   r.TLS != nil, // Secure should be true when serving over HTTPS
		})

		// Respond with user data or a success message
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "logged in",
			"user":   userInfo,
			// Include any other user info you want to return to the client
		})
		if err != nil {
			log.Printf("Error sending response: %v", err)
			http.Error(w, "Failed to send response", http.StatusInternalServerError)
		}
	}
}

func LogoutH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		// Retrieve sessionID from the cookie, assuming you have set it in a cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			log.Printf("Error retrieving session: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		sessionID := cookie.Value

		// Create an empty cookie with expiration time in the past
		expiration := time.Unix(0, 0)
		cookie = &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Expires:  expiration,
			HttpOnly: true,
			Secure:   true, // Set to true if you are using HTTPS
			Path:     "/",
		}

		err = model.DeleteSession(db, sessionID) // sessionID is retrieved from the cookie
		if err != nil {
			log.Printf("Error deleting session: %v", err)
			retryCount := 3
			for i := 1; i <= retryCount; i++ {
				err = model.DeleteSession(db, sessionID)
				if err == nil {
					break
				}
				log.Printf("Retry %d: Error deleting session: %v", i, err)
			}
			if err != nil {
				// Fails after 'retryCount' attempts
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}

		// Set the empty cookie to the HTTP response, effectively deleting the cookie
		http.SetCookie(w, cookie)

		log.Println("User logged out and session ended")
	}
}

func FetchUseH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("Fething users for group creation...")

		users, err := model.FetchAllUsers(db)
		if err != nil {
			log.Printf("Error fetching users: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(users); err != nil {
			log.Printf("Error encoding users to JSON: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func FollowH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		var req struct {
			UserId int    `json:"userId"`
			Action string `json:"action"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
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

		switch req.Action {
		case "follow":
			err = model.FollowUser(db, userID, req.UserId)
		case "unfollow":
			err = model.UnfollowUser(db, userID, req.UserId)
		default:
			http.Error(w, "Invalid Action", http.StatusBadRequest)
			return
		}

		if err != nil {
			log.Printf("Error processing follow action: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}
