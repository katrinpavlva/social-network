package chat

import (
	"database/sql"
	"log"
	"sync"

	"github.com/google/uuid"
)

type Room struct {
	ID      string
	Clients map[*C]bool
	mutex   sync.Mutex
}

func GetCreateRoom(db *sql.DB, user1ID, user2ID int) (string, error) {
	var roomID string

	// SQL query to check if a room already exists
	query := `SELECT RoomID FROM Rooms WHERE (User1ID = ? AND User2ID = ?) OR (User1ID = ? AND User2ID = ?)`
	err := db.QueryRow(query, user1ID, user2ID, user2ID, user1ID).Scan(&roomID)

	if err == sql.ErrNoRows {
		// Room does not exist, create a new one
		roomID = uuid.New().String() // Generate a new UUID

		insertQuery := `INSERT INTO Rooms (RoomID, User1ID, User2ID) VALUES (?, ?, ?)`
		_, err := db.Exec(insertQuery, roomID, user1ID, user2ID)
		if err != nil {
			return "", err
		}
		log.Printf("Created new room with ID: %s", roomID)
	} else if err != nil {
		// An error occurred
		return "", err
	}

	return roomID, nil
}

func (server *WSServer) addToR(client *C, roomID string) {
	room := server.findCreateRoom(roomID)
	room.mutex.Lock() // Lock the mutex
	room.Clients[client] = true
	room.mutex.Unlock() // Unlock the mutex
	client.room = room
	log.Printf("Added client to room: %s", roomID)
}

func (server *WSServer) findCreateRoom(roomID string) *Room {
	if room, ok := server.rooms[roomID]; ok {
		return room
	}
	// Room doesn't exist, so create a new one
	newRoom := &Room{
		ID:      roomID,
		Clients: make(map[*C]bool),
	}
	server.rooms[roomID] = newRoom
	return newRoom
}

// In room.go
func GetCreateGrChatRoom(db *sql.DB, groupId int) (string, error) {
	var roomId string
	// Query to find existing RoomID for the given GroupID
	err := db.QueryRow("SELECT RoomID FROM GroupChatRoom WHERE GroupID = ?", groupId).Scan(&roomId)
	if err == sql.ErrNoRows {
		// Room does not exist, create a new one
		roomId = uuid.New().String() // Generate a new UUID for the room
		_, err := db.Exec("INSERT INTO GroupChatRoom (RoomID, GroupID) VALUES (?, ?)", roomId, groupId)
		if err != nil {
			// Return an empty string and the error if unable to create the room
			return "", err
		}
		// Return the new RoomID and nil as there was no error
		return roomId, nil
	} else if err != nil {
		// Return an empty string and the error if any other error occurred
		return "", err
	}
	// Return the existing RoomID and nil as there was no error
	return roomId, nil
}

// In WSServer methods in ClientServer.go or wherever addToGroupChatRoom is defined
func (server *WSServer) addToGroupChatRoom(db *sql.DB, client *C, groupId int) (string, error) {
	roomId, err := GetCreateGrChatRoom(db, groupId)
	if err != nil {
		log.Printf("Error getting or creating group chat room: %v", err)
		return "", err // return an empty roomId and the error
	}
	server.addToR(client, roomId) // Assuming addToR can handle both private and group chats
	return roomId, nil            // return the roomId and no error
}
