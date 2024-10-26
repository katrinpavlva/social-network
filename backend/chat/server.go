package chat

import (
	"time"
)

type WSServer struct {
	clients    map[*C]bool
	register   chan *C
	unregister chan *C
	broadcast  chan []byte
	rooms      map[string]*Room
}

// NewWSServer creates a new WSServer type
func NewWSServer() *WSServer {
	return &WSServer{
		clients:    make(map[*C]bool),
		register:   make(chan *C),
		unregister: make(chan *C),
		broadcast:  make(chan []byte),
		rooms:      make(map[string]*Room),
	}
}

// Run our websocket server, accepting various requests
func (server *WSServer) Run() {
	for {
		select {

		case client := <-server.register:
			server.registerClient(client)

		case client := <-server.unregister:
			server.unregisterClient(client)
		}

	}
}

func (server *WSServer) registerClient(client *C) {
	server.clients[client] = true
}

func (server *WSServer) unregisterClient(client *C) {
	if _, ok := server.clients[client]; ok {
		delete(server.clients, client)
	}
}

func (room *Room) broadcastToClients(message []byte) {
	room.mutex.Lock()         // Lock the mutex
	defer room.mutex.Unlock() // Unlock the mutex when the function exits

	for client := range room.Clients {
		select {
		case client.send <- message:
			// Message sent successfully
		default:
			// Message queue is full, buffer the message
			client.sendBuffer = append(client.sendBuffer, message)
		}
	}
}

func (client *C) handleBufferedMessages() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Try to send buffered messages
		for len(client.sendBuffer) > 0 {
			select {
			case client.send <- client.sendBuffer[0]:
				// Remove the sent message from the buffer
				client.sendBuffer = client.sendBuffer[1:]
			default:
				// The send channel is still full, try again later
				return
			}
		}
	}
}
