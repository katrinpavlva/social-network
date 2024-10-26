package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"social-network/backend/auth"
	"social-network/backend/model"
)

type GroupCrRequest struct {
	Group          model.Group `json:"group"`
	InvitedUserIds []int       `json:"invitedUserIds"`
}

// CreateGrH handles the creation of a new group.
func CreateGrH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS if needed
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("Inside CreateGrH")

		// Check for the session cookie and retrieve the user ID.
		cookie, err := r.Cookie("session_id")
		if err != nil {
			log.Printf("Error retrieving session: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		sessionID := cookie.Value
		userID, err := model.GetUserIDBySessionID(db, sessionID)
		if err != nil {
			log.Printf("Error retrieving user ID: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Decode the request body into the GroupCrRequest struct.
		var creationReq GroupCrRequest
		if err := json.NewDecoder(r.Body).Decode(&creationReq); err != nil {
			log.Printf("Error decoding group creation request: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Set the creator's user ID to the Group struct.
		creationReq.Group.CreatorUserID = userID

		// Log the incoming group creation data.
		log.Printf("Creating group: %+v", creationReq.Group)
		log.Printf("Inviting user IDs: %+v", creationReq.InvitedUserIds)

		// Create the group and handle user invitations.
		createdGroup, err := model.CreateGroup(db, creationReq.Group, creationReq.InvitedUserIds)
		if err != nil {
			log.Printf("Error creating group: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Println("Group created successfully:", createdGroup)

		// Respond with the newly created group data.
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(createdGroup); err != nil {
			log.Printf("Error sending group response: %v", err)
			http.Error(w, "Error sending group response", http.StatusInternalServerError)
			return
		}
	}
}

func GetGrH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the model function to get the groups
		groups, err := model.GetGroups(db)
		if err != nil {
			log.Printf("Error getting groups: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Send the groups as a JSON response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(groups); err != nil {
			log.Printf("Error encoding groups response: %v", err)
			http.Error(w, "Error sending groups response", http.StatusInternalServerError)
			return
		}
	}
}

func FetchGrDetailH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		var requestData struct {
			GroupID string `json:"groupId"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		log.Printf("Fetching details for group ID: %s", requestData.GroupID)

		// Call a function to fetch the group details
		group, err := model.GetGroupByID(db, requestData.GroupID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with the group details
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(group); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func FetchGrMemH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Extract groupID from the request URL
		groupID, err := strconv.Atoi(r.URL.Query().Get("groupID"))
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		members, err := model.GetGroupMembers(db, groupID)
		if err != nil {
			http.Error(w, "Failed to fetch group members", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(members)
	}
}

func CreateEvH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("Inside CreateEvH")

		// Decode the request body into the EventCreationRequest struct.
		var creationReq model.EventCreationRequest
		if err := json.NewDecoder(r.Body).Decode(&creationReq); err != nil {
			log.Printf("Error decoding event creation request: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Log the incoming event creation data.
		log.Printf("Creating event: %+v", creationReq)

		// Create the event and handle invitations.
		event, err := model.CreateEvent(db, creationReq)
		if err != nil {
			log.Printf("Error creating event: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Printf("Event created successfully: %+v", event)

		// Respond with the newly created event data.
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(event); err != nil {
			log.Printf("Error sending event response: %v", err)
			http.Error(w, "Error sending event response", http.StatusInternalServerError)
			return
		}
	}
}

func GetEvH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		groupID := r.URL.Query().Get("groupID")
		log.Printf("Fetching events for GroupID: %s", groupID) // Log the GroupID

		if groupID == "" {
			http.Error(w, "Group ID is required", http.StatusBadRequest)
			return
		}

		events, err := model.GetGroupEvents(db, groupID)
		if err != nil {
			log.Printf("Error fetching events for group %s: %v", groupID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Printf("Fetched %d events for GroupID: %s", len(events), groupID) // Log the number of events fetched

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events)
	}
}

func JoinGrH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("Inside JoinGrH")

		var joinReq model.GroupJoinRequest
		if err := json.NewDecoder(r.Body).Decode(&joinReq); err != nil {
			log.Printf("Error decoding join group request: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		log.Printf("Request to join group: %+v", joinReq)

		err := model.JoinGroup(db, joinReq)
		if err != nil {
			log.Printf("Error processing join group request: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Println("Join group request processed successfully")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Join group request sent"})
	}
}

func LeaveGrH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		var leaveReq model.GroupLeaveRequest
		if err := json.NewDecoder(r.Body).Decode(&leaveReq); err != nil {
			log.Printf("Error decoding leave group request: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		err := model.LeaveGroup(db, leaveReq)
		if err != nil {
			log.Printf("Error processing leave group request: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Println("Leave group request processed successfully")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Leave group request sent"})

	}
}

func InviteUserH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		var invitationRequest struct {
			GroupID        int   `json:"groupId"`
			InvitedUserIds []int `json:"invitedUserIds"`
		}

		log.Printf("Received invitation request: %+v\n", r.Body)

		if err := json.NewDecoder(r.Body).Decode(&invitationRequest); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		log.Printf("Received invitation request: %+v\n", invitationRequest)

		if err := model.InviteUsersToGroup(db, invitationRequest.GroupID, invitationRequest.InvitedUserIds); err != nil {
			log.Printf("Error inviting users to group: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invitations sent successfully"})
	}
}

func GetInvUserH(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.EnableCors(&w)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		// Define a struct to decode the request body
		var req struct {
			GroupID int `json:"groupId"`
		}

		// Decode the JSON body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Use req.GroupID to fetch invited users
		invitedUsers, err := model.GetInvitedUsers(db, req.GroupID)
		if err != nil {
			http.Error(w, "Failed to fetch invited users", http.StatusInternalServerError)
			return
		}

		// Respond with the list of invited users
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(invitedUsers); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}
