package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"social-network/backend/model"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Max wait time when writing message to peer
	writeWait = 10 * time.Second

	// Max time till next pong from peer
	pongWait = 60 * time.Second

	// Send ping interval, must be less then pong wait time
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10000
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// C represents the websocket client at the server
type C struct {
	conn       *websocket.Conn
	wsServer   *WSServer
	send       chan []byte
	room       *Room
	sendBuffer []json.RawMessage
}

type URelation struct {
	Nickname       string
	RoomID         string
	UserID         int
	UnreadCount    int
	FirstName      string
	LastName       string
	ProfilePicture string `json:"profilePicture"`
}

type SockMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type IncomingMessage struct {
	SenderUserID   int       `json:"senderUserId"`
	ReceiverUserID int       `json:"receiverUserId,omitempty"`
	RoomID         string    `json:"roomId"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
	GroupID        int       `json:"groupId,omitempty"`
}

type Message struct {
	MessageID         string    `json:"messageId"`
	SenderUserID      int       `json:"senderUserId"`
	ReceiverUserID    int       `json:"receiverUserId"`
	RoomID            string    `json:"roomId"`
	Content           string    `json:"content"`
	Timestamp         time.Time `json:"timestamp"`
	Read              bool      `json:"read"`
	SenderFirstName   string    `json:"senderFirstName"`
	SenderLastName    string    `json:"senderLastName"`
	SenderNickname    string    `json:"senderNickname"`
	ReceiverFirstName string    `json:"receiverFirstName"`
	ReceiverLastName  string    `json:"receiverLastName"`
	ReceiverNickname  string    `json:"receiverNickname"`
}

type EvResponseNotification struct {
	EventID          int       `json:"eventId"`
	GroupID          int       `json:"groupId"`
	GroupName        string    `json:"groupName"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	EventDateTime    time.Time `json:"eventDateTime"`
	CreatorID        int       `json:"creatorId"`
	CreatorFirstName string    `json:"creatorFirstName"`
	CreatorLastName  string    `json:"creatorLastName"`
	CreatedAt        time.Time `json:"createdAt"`
	ResponseID       int       `json:"responseId"`
	UserID           int       `json:"userId"`
	Response         *string   `json:"response"`
}

type GroupInvite struct {
	GroupID       int
	Name          string
	Description   string
	CreatorUserID int
	CreatorName   string
}

type FollowRequest struct {
	FollowerUserID int    `json:"followerUserId"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
}

type GrJoinReqNotification struct {
	RequestId int    `json:"requestId"`
	UserId    int    `json:"userId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	GroupId   int    `json:"groupId"`
	GroupName string `json:"groupName"`
}

func newClient(conn *websocket.Conn, wsServer *WSServer) *C {
	return &C{
		conn:     conn,
		wsServer: wsServer,
		send:     make(chan []byte, 256),
	}

}

func (client *C) readPump(db *sql.DB, UserID int) {
	log.Println("Starting readPump for client")
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in readPump: %v", r)
		}
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error { client.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// Start endless read loop, waiting for messages from client
	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		log.Println("Received a new message from client")

		var wsMessage SockMessage
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Println("Error unmarshaling WebSocket message:", err)
			continue
		}

		switch wsMessage.Type {
		// Add a new case for checking notifications
		case "eventInvite":
			log.Printf("Raw message received: %s", message)
			// Extract the user ID from the payload if needed
			var notificationCheck struct {
				UserID int `json:"userId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &notificationCheck); err != nil {
				log.Println("Error unmarshaling notification check request:", err)
				continue
			}

			// Log the notification check request
			log.Printf("Checking notifications for user ID: %d", notificationCheck.UserID)

			// Query the datab for notifications
			notifications, err := CheckEventInvite(db, notificationCheck.UserID)
			if err != nil {
				log.Println("Error checking notifications:", err)
				// Send an error response if needed
				continue
			}

			// Prepare the response
			var response SockMessage
			if len(notifications) == 0 {
				// No new notifications
				response = SockMessage{
					Type:    "eventInviteResponse",
					Payload: json.RawMessage(`{"message": "No New Notifications"}`),
				}
			} else {
				// New notifications found
				notificationsPayload, _ := json.Marshal(notifications)
				response = SockMessage{
					Type:    "eventInviteResponse",
					Payload: json.RawMessage(notificationsPayload),
				}
			}

			// Send the response back to the client
			responseJSON, _ := json.Marshal(response)
			client.send <- responseJSON

			log.Println("Notification check response sent to client")

		case "eInviteResponse":
			var eventResponse struct {
				ResponseId int    `json:"responseId"`
				UserId     int    `json:"userId"`
				Response   string `json:"response"` // going or notGoing
			}
			if err := json.Unmarshal(wsMessage.Payload, &eventResponse); err != nil {
				log.Println("Error unmarshaling event response:", err)
				continue
			}

			// Process the event response here (e.g., update the datab with the user's response)
			err := ProcessEventResponse(db, eventResponse.ResponseId, eventResponse.UserId, eventResponse.Response)
			if err != nil {
				// Handle error
				log.Println("Error processing event response:", err)
			}

		case "groupInvite":
			log.Printf("Received groupInvite check request: %s", message)
			// Extract the user ID from the payload
			var groupInviteCheck struct {
				UserID int `json:"userId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &groupInviteCheck); err != nil {
				log.Printf("Error unmarshaling groupInvite check request: %v", err)
				// Send an error response if needed
				continue
			}

			// Log the group invite check request
			log.Printf("Checking group invites for user ID: %d", groupInviteCheck.UserID)

			// Query the datab for group invites
			invites, err := CheckGroupInvites(db, groupInviteCheck.UserID)
			if err != nil {
				log.Printf("Error checking group invites: %v", err)
				// Send an error response if needed
				continue
			}

			// Prepare the response
			var response SockMessage
			if len(invites) == 0 {
				// No new group invites
				response = SockMessage{
					Type:    "groupInviteResponse",
					Payload: json.RawMessage(`{"message": "No New Group Invites"}`),
				}
			} else {
				// New group invites found
				invitesPayload, _ := json.Marshal(invites)
				response = SockMessage{
					Type:    "groupInviteResponse",
					Payload: json.RawMessage(invitesPayload),
				}
			}

			// Send the response back to the client
			responseJSON, _ := json.Marshal(response)
			client.send <- responseJSON

			log.Printf("Group invite check response sent to client")

		case "gInviteResponse":
			var groupResponse struct {
				GroupID int  `json:"groupId"`
				UserID  int  `json:"userId"`
				Accept  bool `json:"accept"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &groupResponse); err != nil {
				log.Println("Error unmarshaling group invite response:", err)
				continue
			}

			// Process the group invite response
			err := HandleGroupInviteResponse(db, groupResponse.UserID, groupResponse.GroupID, groupResponse.Accept)
			if err != nil {
				log.Println("Error processing group invite response:", err)
			}

		case "followRequest":
			var followReq struct {
				TargetUserId    int `json:"targetUserId"`
				RequesterUserId int `json:"requesterUserId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &followReq); err != nil {
				log.Println("Error unmarshaling follow request:", err)
				continue
			}

			// Save the follow request to the datab
			err := SaveFollowRequest(db, followReq.RequesterUserId, followReq.TargetUserId)
			if err != nil {
				log.Println("Error saving follow request:", err)
				// Optionally, send a failure response back to the client
				continue
			}

		case "acceptFollowRequest":
			var payload struct {
				UserId         int `json:"userId"`
				FollowerUserId int `json:"followerUserId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &payload); err != nil {
				log.Println("Error unmarshaling accept follow request:", err)
				continue
			}

			// Accept the follow request
			err := AcceptFollowRequest(db, payload.FollowerUserId, payload.UserId)
			if err != nil {
				log.Println("Error accepting follow request:", err)
				// Optionally send an error response back to the client
			}

		case "declineFollowRequest":
			var payload struct {
				UserId         int `json:"userId"`
				FollowerUserId int `json:"followerUserId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &payload); err != nil {
				log.Println("Error unmarshaling decline follow request:", err)
				continue
			}

			// Decline the follow request
			err := RemoveFollowRequest(db, payload.FollowerUserId, payload.UserId)
			if err != nil {
				log.Println("Error declining follow request:", err)
			}

		case "cancelFollowRequest":
			var cancelPayload struct {
				TargetUserId    int `json:"targetUserId"`
				RequesterUserId int `json:"requesterUserId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &cancelPayload); err != nil {
				log.Println("Error unmarshaling cancel follow request:", err)
				continue
			}

			err := RemoveFollowRequest(db, cancelPayload.RequesterUserId, cancelPayload.TargetUserId)
			if err != nil {
				log.Println("Error removing follow request:", err)
			}

		case "followRequestCheck":
			var checkPayload struct {
				UserId int `json:"userId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &checkPayload); err != nil {
				log.Println("Error unmarshaling follow request check:", err)
				continue
			}

			// Fetch follow requests
			followRequests, err := FetchFollowRequests(db, checkPayload.UserId)
			followRequestsJSON, err := json.Marshal(followRequests)
			log.Printf("Marshalled follow requests JSON: %s", string(followRequestsJSON))

			if err != nil {
				log.Println("Error marshaling follow requests:", err)
				// Handle error appropriately
			} else {
				response := SockMessage{
					Type:    "followRequestResponse",
					Payload: json.RawMessage(followRequestsJSON),
				}
				responseJSON, _ := json.Marshal(response)
				client.send <- responseJSON
			}

		case "groupJoinRequestCheck":
			var checkPayload struct {
				UserId int `json:"userId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &checkPayload); err != nil {
				log.Println("Error unmarshaling group join request check:", err)
				continue
			}

			requests, err := FetchGroupJoinRequests(db, checkPayload.UserId)
			if err != nil {
				log.Printf("Error fetching group join requests: %v", err)
				continue
			}

			responsePayload, err := json.Marshal(requests)
			if err != nil {
				log.Printf("Error marshaling group join requests response: %v", err)
				continue
			}

			response := SockMessage{
				Type:    "groupJoinRequestResponse",
				Payload: json.RawMessage(responsePayload),
			}
			responseJSON, _ := json.Marshal(response)
			client.send <- responseJSON

		case "acceptGroupJoinRequest":
			var acceptPayload struct {
				RequestId int `json:"requestId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &acceptPayload); err != nil {
				log.Println("Error unmarshaling accept group join request:", err)
				continue
			}

			// Accept the group join request logic
			err := AcceptGroupJoinRequest(db, acceptPayload.RequestId, UserID)
			if err != nil {
				log.Println("Error accepting group join request:", err)
				// Optionally send an error response back to the client
			}

		case "declineGroupJoinRequest":
			var declinePayload struct {
				RequestId int `json:"requestId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &declinePayload); err != nil {
				log.Println("Error unmarshaling decline group join request:", err)
				continue
			}

			// Decline the group join request logic
			err := DeclineGroupJoinRequest(db, declinePayload.RequestId)
			if err != nil {
				log.Println("Error declining group join request:", err)
				// Optionally send an error response back to the client
			}

		case "chatMessage":
			var chatMsg IncomingMessage
			if err := json.Unmarshal(wsMessage.Payload, &chatMsg); err != nil {
				log.Println("Error unmarshaling chat message:", err)
				continue
			}

			// Fetch sender's first name and last name
			firstName, lastName, err := model.GetUserDetails(db, chatMsg.SenderUserID)
			if err != nil {
				log.Println("Failed to fetch user details for sender:", err)
				continue // Or handle the error as you see fit
			}

			log.Printf("Received a chat message: %+v", chatMsg)

			// Save the message to the datab
			if err = saveMessage(db, chatMsg); err != nil {
				log.Println("Error saving message:", err)
				continue
			}

			// Construct the broadcast message including sender's name
			broadcastMessage := SockMessage{
				Type: "chatMessage",
				Payload: json.RawMessage(fmt.Sprintf(`{
					"senderUserId": %d,
					"senderFirstName": "%s",
					"senderLastName": "%s",
					"content": "%s",
					"roomId": "%s",
					"timestamp": "%s",
					"groupId": %d
				}`, chatMsg.SenderUserID, firstName, lastName, chatMsg.Content, chatMsg.RoomID, chatMsg.Timestamp.Format(time.RFC3339), chatMsg.GroupID)),
			}

			broadcastJSON, _ := json.Marshal(broadcastMessage)

			// Broadcast the message to the room
			if room, ok := client.wsServer.rooms[chatMsg.RoomID]; ok {
				log.Printf("Broadcasting message to room: %s", chatMsg.RoomID)
				room.broadcastToClients(broadcastJSON)
			}

			log.Printf("Message broadcasted to room: %s", chatMsg.RoomID)

		case "joinGroupChat":
			var joinMsg struct {
				GroupId string `json:"groupId"`
			}
			if err := json.Unmarshal(wsMessage.Payload, &joinMsg); err != nil {
				log.Println("Error unmarshaling join group chat message:", err)
				continue
			}

			log.Printf("C requests to join group chat: %s", joinMsg.GroupId)
			// Convert GroupId to int before passing
			groupIdInt, err := strconv.Atoi(joinMsg.GroupId)
			if err != nil {
				log.Printf("Error converting GroupId to int: %v", err)
				continue
			}
			roomId, err := client.wsServer.addToGroupChatRoom(db, client, groupIdInt)
			if err != nil {
				log.Printf("Error joining group chat room: %v", err)
				// Optionally send an error message back to the client
				// Ensure you construct and send a proper error response here if desired
			} else {
				// Send a response back to the client with the roomId
				response := SockMessage{
					Type:    "joinGroupChatResponse",
					Payload: json.RawMessage(fmt.Sprintf(`{"roomId": "%s"}`, roomId)),
				}
				responseJSON, err := json.Marshal(response)
				if err != nil {
					log.Println("Error marshaling join group chat response:", err)
					// Optionally handle the error, e.g., by logging or sending an error message to the client
					continue
				}
				log.Printf("Sending joinGroupChatResponse: %s", string(responseJSON))
				client.send <- responseJSON
			}

		case "fetchMessages":
			var fetchRequest struct {
				RoomID  string `json:"roomId"`
				GroupID *int   `json:"groupId,omitempty"` // Use pointer to detect if groupId was provided
			}
			if err := json.Unmarshal(wsMessage.Payload, &fetchRequest); err != nil {
				log.Println("Error unmarshaling fetch messages request:", err)
				continue
			}

			log.Printf("Received a fetch messages request: %+v", fetchRequest)

			messages, err := FetchMessages(db, fetchRequest.RoomID, UserID, fetchRequest.GroupID)
			if err != nil {
				log.Println("Error fetching messages:", err)
				continue
			}

			responsePayload := map[string]interface{}{
				"type": "fetchMessagesResponse",
				"payload": map[string]interface{}{
					"roomId":   fetchRequest.RoomID,
					"messages": messages,
				},
			}

			responseJSON, err := json.Marshal(responsePayload)
			if err != nil {
				log.Println("Error marshaling fetch messages response:", err)
				continue
			}

			client.send <- responseJSON

		}
	}
}

func saveMessage(db *sql.DB, message IncomingMessage) error {
	var query string
	var args []interface{}
	messageID := uuid.New().String()

	if message.GroupID != 0 {
		// This is a group chat message
		query = `INSERT INTO GroupChatMessage (MessageID, GroupID, SenderUserID, Content, RoomID) VALUES (?, ?, ?, ?, ?)` // Corrected: added a placeholder for RoomID
		args = []interface{}{messageID, message.GroupID, message.SenderUserID, message.Content, message.RoomID}
	} else {
		// This is a private chat message
		query = `INSERT INTO Message (MessageID, SenderUserID, ReceiverUserID, RoomID, Content) VALUES (?, ?, ?, ?, ?)` // Already correct
		args = []interface{}{messageID, message.SenderUserID, message.ReceiverUserID, message.RoomID, message.Content}
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		log.Println("Error saving message:", err)
		return err
	}
	return nil
}

func FetchMessages(db *sql.DB, roomID string, userID int, groupID *int) ([]Message, error) {
	var messages []Message
	var query string

	if groupID != nil {
		// Fetch from GroupChatMessage if groupId is provided
		query = `
			SELECT m.MessageID, m.RoomID, m.Content, m.Timestamp, 
			s.UserID AS SenderUserID, s.FirstName AS SenderFirstName, s.LastName AS SenderLastName, s.Nickname AS SenderNickname
			FROM GroupChatMessage m
			JOIN User s ON m.SenderUserID = s.UserID
			WHERE m.RoomID = ? AND m.GroupID = ?
			ORDER BY m.Timestamp DESC`
	} else {
		// Fetch from Message if groupId is not provided
		query = `
			SELECT m.MessageID, m.RoomID, m.Content, m.Timestamp, m.Read, 
			s.UserID AS SenderUserID, s.FirstName AS SenderFirstName, s.LastName AS SenderLastName, s.Nickname AS SenderNickname,
			r.UserID AS ReceiverUserID, r.FirstName AS ReceiverFirstName, r.LastName AS ReceiverLastName, r.Nickname AS ReceiverNickname
			FROM Message m
			JOIN User s ON m.SenderUserID = s.UserID
			JOIN User r ON m.ReceiverUserID = r.UserID
			WHERE m.RoomID = ?
			ORDER BY m.Timestamp DESC`
	}

	var rows *sql.Rows
	var err error

	if groupID != nil {
		rows, err = db.Query(query, roomID, *groupID)
	} else {
		rows, err = db.Query(query, roomID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var message Message
		// Adjust the Scan based on the query being executed
		if groupID != nil {
			err := rows.Scan(&message.MessageID, &message.RoomID, &message.Content, &message.Timestamp,
				&message.SenderUserID, &message.SenderFirstName, &message.SenderLastName, &message.SenderNickname)
			if err != nil {
				return nil, err
			}
		} else {
			err := rows.Scan(&message.MessageID, &message.RoomID, &message.Content, &message.Timestamp, &message.Read,
				&message.SenderUserID, &message.SenderFirstName, &message.SenderLastName, &message.SenderNickname,
				&message.ReceiverUserID, &message.ReceiverFirstName, &message.ReceiverLastName, &message.ReceiverNickname)
			if err != nil {
				return nil, err
			}
		}
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	updateQuery := `UPDATE Message SET Read = TRUE WHERE RoomID = ? AND ReceiverUserID = ? AND Read = FALSE`
	_, err = db.Exec(updateQuery, roomID, userID)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func CheckEventInvite(db *sql.DB, userID int) ([]EvResponseNotification, error) {
	var notifications []EvResponseNotification
	query := `
	SELECT 
		e.EventID, c.Name as GroupName, e.GroupID, e.Title, e.Description, e.EventDateTime, e.CreatorID, u.FirstName, u.LastName, e.CreatedAt,
		uer.ResponseID, uer.UserID, IFNULL(uer.Response, '') as Response
	FROM 
		Event e
	INNER JOIN 
		UserEventResponse uer ON e.EventID = uer.EventID
	INNER JOIN 
		User u ON e.CreatorID = u.UserID
	INNER JOIN 
		Cluster c ON e.GroupID = c.GroupID
	WHERE 
		uer.UserID = ? AND (uer.Response IS NULL OR uer.Response = '')
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying event invites: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var notification EvResponseNotification
		var response sql.NullString // Use sql.NullString to handle NULL values
		if err := rows.Scan(&notification.EventID, &notification.GroupName, &notification.GroupID, &notification.Title, &notification.Description, &notification.EventDateTime, &notification.CreatorID, &notification.CreatorFirstName, &notification.CreatorLastName, &notification.CreatedAt, &notification.ResponseID, &notification.UserID, &response); err != nil {
			return nil, fmt.Errorf("scanning event invite: %v", err)
		}
		// Convert sql.NullString to *string
		if response.Valid {
			notification.Response = &response.String
		} else {
			var noResponse *string = nil // Represents a NULL response
			notification.Response = noResponse
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

func ProcessEventResponse(db *sql.DB, responseID, userID int, response string) error {
	statement := `UPDATE UserEventResponse SET Response = ? WHERE ResponseID = ? AND UserID = ?`
	_, err := db.Exec(statement, response, responseID, userID)
	if err != nil {
		return fmt.Errorf("updating event response: %v", err)
	}
	return nil
}

func CheckGroupInvites(db *sql.DB, userID int) ([]GroupInvite, error) {
	var invites []GroupInvite
	query := `
			SELECT c.GroupID, c.Name, c.Description, c.CreatorUserID, u.FirstName, u.LastName
			FROM InvitedUsers i
			JOIN Cluster c ON i.GroupID = c.GroupID
			JOIN User u ON c.CreatorUserID = u.UserID
			WHERE i.UserID = ? AND i.Accepted = 0
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error querying group invites: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var invite GroupInvite
		var firstName, lastName string
		if err := rows.Scan(&invite.GroupID, &invite.Name, &invite.Description, &invite.CreatorUserID, &firstName, &lastName); err != nil {
			log.Printf("Error scanning group invite: %v", err)
			continue
		}
		invite.CreatorName = firstName + " " + lastName // Concatenate first name and last name
		invites = append(invites, invite)
	}

	return invites, nil
}

func HandleGroupInviteResponse(db *sql.DB, userID, groupID int, accept bool) error {
	if accept {
		// If the user accepted the invitation, add them to the GroupMembers table
		_, err := db.Exec(`INSERT INTO GroupMembers (GroupID, UserID, Accepted) VALUES (?, ?, TRUE)`, groupID, userID)
		if err != nil {
			log.Printf("Error adding user to GroupMembers: %v", err)
			return err
		}
	}
	// Remove the invitation from InvitedUsers regardless of accept or decline
	_, err := db.Exec(`DELETE FROM InvitedUsers WHERE GroupID = ? AND UserID = ?`, groupID, userID)
	if err != nil {
		log.Printf("Error removing invitation: %v", err)
		return err
	}

	return nil
}

func FetchFollowRequests(db *sql.DB, userId int) ([]FollowRequest, error) {
	var requests []FollowRequest
	query := `
	SELECT fr.FollowerUserID, u.FirstName, u.LastName
	FROM FollowRequests fr
	JOIN User u ON fr.FollowerUserID = u.UserID
	WHERE fr.FollowingUserID = ? AND fr.Accepted IS FALSE
	`
	rows, err := db.Query(query, userId)
	if err != nil {
		log.Printf("Error executing follow requests fetch query: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var req FollowRequest
		if err := rows.Scan(&req.FollowerUserID, &req.FirstName, &req.LastName); err != nil {
			log.Printf("Error scanning follow request: %v", err)
			continue // or return nil, err if you want to stop processing on first error
		}
		requests = append(requests, req)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating follow requests rows: %v", err)
		return nil, err
	}

	log.Printf("%d follow requests fetched for user %d", len(requests), userId)

	// Log marshalled JSON for debugging purposes
	if marshalledJSON, err := json.Marshal(requests); err != nil {
		log.Printf("Error marshalling follow requests to JSON: %v", err)
	} else {
		log.Printf("Marshalled follow requests JSON: %s", string(marshalledJSON))
	}

	if len(requests) == 0 {
		log.Printf("No follow requests found for user %d", userId)
		// You might choose to return an empty slice instead of nil to explicitly indicate no results found
		return requests, nil
	}

	return requests, nil
}

func AcceptFollowRequest(db *sql.DB, followerUserID, followingUserID int) error {
	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Insert the follow relationship into UserFollowers
	_, err = tx.Exec(`
			INSERT INTO UserFollowers (FollowerUserID, FollowingUserID)
			VALUES (?, ?)`,
		followerUserID, followingUserID)
	if err != nil {
		tx.Rollback() // Rollback in case of error
		return err
	}

	// Delete the follow request from FollowRequests
	_, err = tx.Exec(`
			DELETE FROM FollowRequests 
			WHERE FollowerUserID = ? AND FollowingUserID = ?`,
		followerUserID, followingUserID)
	if err != nil {
		tx.Rollback() // Rollback in case of error
		return err
	}

	// Commit the transaction
	return tx.Commit()
}

func SaveFollowRequest(db *sql.DB, followerUserID, followingUserID int) error {
	_, err := db.Exec(`
			INSERT INTO FollowRequests (FollowerUserID, FollowingUserID)
			VALUES (?, ?)
	`, followerUserID, followingUserID)

	if err != nil {
		log.Printf("Error saving follow request: %v", err)
		return err
	}
	return nil
}

func RemoveFollowRequest(db *sql.DB, requesterUserId, targetUserId int) error {
	_, err := db.Exec("DELETE FROM FollowRequests WHERE FollowerUserID = ? AND FollowingUserID = ?", requesterUserId, targetUserId)

	if err != nil {
		log.Printf("Error saving follow request: %v", err)
		return err
	}
	return err
}

func FetchGroupJoinRequests(db *sql.DB, groupCreatorId int) ([]GrJoinReqNotification, error) {
	var requests []GrJoinReqNotification
	query := `
	SELECT gjr.RequestId, gjr.UserId, u.FirstName, u.LastName, c.GroupId, c.Name
	FROM GroupJoinRequests gjr
	JOIN User u ON gjr.UserId = u.UserID
	JOIN Cluster c ON gjr.GroupId = c.GroupID
	WHERE c.CreatorUserID = ?
	`
	rows, err := db.Query(query, groupCreatorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var request GrJoinReqNotification
		if err := rows.Scan(&request.RequestId, &request.UserId, &request.FirstName, &request.LastName, &request.GroupId, &request.GroupName); err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func AcceptGroupJoinRequest(db *sql.DB, requestId int, userId int) error {
	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Find the group join request details
	var groupId, requesterId int
	err = tx.QueryRow("SELECT GroupId, UserId FROM GroupJoinRequests WHERE RequestId = ?", requestId).Scan(&groupId, &requesterId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert into GroupMembers
	_, err = tx.Exec("INSERT INTO GroupMembers (GroupID, UserID, Accepted) VALUES (?, ?, TRUE)", groupId, requesterId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete from GroupJoinRequests
	_, err = tx.Exec("DELETE FROM GroupJoinRequests WHERE RequestId = ?", requestId)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func DeclineGroupJoinRequest(db *sql.DB, requestId int) error {
	_, err := db.Exec("DELETE FROM GroupJoinRequests WHERE RequestId = ?", requestId)
	return err
}

func (client *C) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()
	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The WSServer closed the channel.
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Attach queued chat messages to the current websocket message.
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *C) disconnect() {
	if client == nil {
		return
	}

	if client.room != nil {
		delete(client.room.Clients, client)
	}
	if client.wsServer != nil {
		client.wsServer.unregister <- client
	}
	close(client.send)
	if client.conn != nil {
		client.conn.Close()
	}
}

// ServeWs handles websocket requests from clients requests.
func ServeWs(db *sql.DB, wsServer *WSServer, w http.ResponseWriter, r *http.Request) {
	log.Println("ServeWs called")

	sessionID, err := r.Cookie("session_id")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println("Failed to get session ID from cookie")
		return
	}
	log.Println("Got session ID from cookie")

	userID, err := model.GetUserIDBySessionID(db, sessionID.Value)
	if err != nil {
		log.Println("Failed to get user ID by session ID:", err)
		return
	}
	log.Printf("Mapped session ID to user ID: %d", userID)

	// Fetch user relations
	userRelations, err := model.GetUserFollowRelations(db, userID)
	if err != nil {
		log.Printf("Error retrieving user relations: %v", err)
		return
	}

	followingMap, followersMap, err := model.GetFollowRelationships(db)
	if err != nil {
		log.Printf("Error retrieving follow relationships: %v", err)
		return
	}

	pendingRequests, err := model.GetPendingFollowRequests(db, userID)
	if err != nil {
		log.Printf("Error retrieving pending requests: %v", err)
		return
	}

	userGroups, err := model.GetUserGroupMemberships(db, userID)
	if err != nil {
		log.Printf("Error retrieving user groups: %v", err)
		return
	}

	groupJoinRequests, err := model.GetUserGroupJoinRequests(db, userID)
	if err != nil {
		log.Printf("Error retrieving group join requests: %v", err)
		return
	}

	updatedRelations := make(map[int]URelation)
	for relatedUserID, relation := range userRelations {
		roomID, err := GetCreateRoom(db, userID, relatedUserID)
		if err != nil {
			log.Printf("Error getting or creating room for users %d and %d: %v", userID, relatedUserID, err)
			continue
		}
		updatedRelations[relatedUserID] = URelation{
			Nickname:       relation.Nickname,
			FirstName:      relation.FirstName,
			LastName:       relation.LastName,
			RoomID:         roomID,
			UserID:         relatedUserID,
			UnreadCount:    relation.UnreadCount,
			ProfilePicture: relation.ProfilePicture,
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, wsServer)

	// Add the user to each room related to their follow relations
	for _, relation := range updatedRelations {
		wsServer.addToR(client, relation.RoomID)
		log.Printf("User %d connected to room %s", userID, relation.RoomID)
	}
	// Wrap the relations data in an object with 'followRelations' key
	initialDataWrapper := map[string]interface{}{
		"userRelations":     updatedRelations,
		"followingMap":      followingMap,
		"followersMap":      followersMap,
		"pendingRequests":   pendingRequests,
		"userGroups":        userGroups,
		"groupJoinRequests": groupJoinRequests,
	}
	initialData, err := json.Marshal(initialDataWrapper)
	if err != nil {
		log.Println("Error marshaling initial data:", err)
		return
	}

	// Send initial data right after WebSocket upgrade
	client.send <- initialData

	go client.writePump()
	go client.readPump(db, userID)
	go client.handleBufferedMessages()

	wsServer.register <- client
}
