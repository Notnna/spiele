package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const (
	maxClients = 2
)

type Room struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan BroadcastMessage
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	maxClients int
}

type BroadcastMessage struct {
	message []byte
	sender  *websocket.Conn
	msgType string
}

type Categories struct {
	Categories []string `json:"categories"`
}

type Server struct {
	rooms      map[string]*Room
	mu         sync.Mutex
	categories []string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

func NewServer() *Server {
	server := &Server{
		rooms: make(map[string]*Room),
	}
	server.loadCategories()
	return server
}

func (s *Server) loadCategories() {
	data, err := ioutil.ReadFile("./app/data/categories.json")
	if err != nil {
		log.Fatalf("Error reading categories file: %v", err)
	}

	var categories Categories
	err = json.Unmarshal(data, &categories)
	if err != nil {
		log.Fatalf("Error unmarshalling categories: %v", err)
	}

	s.categories = categories.Categories
	log.Printf("Loaded %d categories", len(s.categories))
}

func (s *Server) getRandomCategory() string {
	return s.categories[rand.Intn(len(s.categories))]
}

func NewRoom() *Room {
	return &Room{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan BroadcastMessage),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		maxClients: maxClients,
	}
}

func (s *Server) getOrCreateRoom(roomID string) (*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		room = NewRoom()
		s.rooms[roomID] = room
		go room.run()
	}
	return room, nil
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}
	defer conn.Close()

	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		log.Println("Error: Room ID is required")
		return
	}

	room, err := s.getOrCreateRoom(roomID)
	if err != nil {
		log.Printf("Error getting or creating room: %v", err)
		return
	}

	// Check if the room is full before registering
	if len(room.clients) >= room.maxClients {
		log.Printf("Room %s is full. Connection rejected.", roomID)
		conn.Close()
		return
	}

	// Register the connection to the room
	room.register <- conn
	log.Printf("New client connected to room: %s", roomID)

	// Handle messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			room.unregister <- conn
			break
		}
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		switch msg["type"] {
		case "newCategory":
			newCategory := s.getRandomCategory()
			newCategoryMsg, err := json.Marshal(map[string]interface{}{
				"type":  "newCategory",
				"value": newCategory,
			})
			if err != nil {
				log.Printf("Error marshalling new category message: %v", err)
				continue
			}
			room.broadcast <- BroadcastMessage{
				message: newCategoryMsg,
				sender:  conn,
				msgType: "newCategory",
			}
		default:
			room.broadcast <- BroadcastMessage{message: message, sender: conn, msgType: msg["type"].(string)}
		}
	}
}

func (r *Room) run() {
	for {
		select {
		case client := <-r.register:
			if len(r.clients) < r.maxClients {
				r.clients[client] = true
				log.Printf("Client registered. Total clients: %d", len(r.clients))
			} else {
				log.Println("Room is full. Rejecting new client.")
				client.Close()
			}
		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				client.Close()
				log.Printf("Client unregistered. Total clients: %d", len(r.clients))
			}
		case broadcastMsg := <-r.broadcast:
			for client := range r.clients {

				if broadcastMsg.msgType != "newCategory" && client == broadcastMsg.sender {
					continue
				}
				err := client.WriteMessage(websocket.TextMessage, broadcastMsg.message)
				if err != nil {
					log.Printf("Error broadcasting message: %v", err)
					client.Close()
					delete(r.clients, client)
				}
			}
		}
	}
}

func (s *Server) cleanupEmptyRooms() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, room := range s.rooms {
		if len(room.clients) == 0 {
			close(room.broadcast)
			close(room.register)
			close(room.unregister)
			delete(s.rooms, id)
			log.Printf("Cleaned up empty room: %s", id)
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	server := NewServer()

	corsHandler := cors.Default().Handler(http.HandlerFunc(server.handleConnections))
	http.Handle("/", http.FileServer(http.Dir("./app/dist")))

	http.Handle("/ws", corsHandler)

	log.Print("WebSocket server starting on 0.0.0.0:8080")
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

	// Periodically clean up empty rooms
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			server.cleanupEmptyRooms()
		}
	}()
}
