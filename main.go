package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed client/dist
var dist embed.FS

//go:embed data
var data embed.FS

const (
	maxClients = 2
)

type Config struct {
	Port            string        `json:"port"`
	MaxClients      int           `json:"maxClients"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	RoomTimeout     time.Duration `json:"roomTimeout"`
	ReadTimeout     time.Duration `json:"readTimeout"`
	WriteTimeout    time.Duration `json:"writeTimeout"`
}

type Room struct {
	clients        map[*websocket.Conn]bool
	broadcast      chan BroadcastMessage
	register       chan *websocket.Conn
	unregister     chan *websocket.Conn
	maxClients     int
	usedCategories []string
	revealed       int
	lastActivity   time.Time
	done           chan struct{}
	server         *Server
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
	distFS     fs.FS
	config     Config
	metrics    *Metrics
	shutdown   chan struct{}
}

type Metrics struct {
	activeRooms   int64
	activeClients int64
	messagesTotal int64
	errorCount    int64
	mu            sync.Mutex
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

func NewServer(config Config) *Server {
	server := &Server{
		rooms:    make(map[string]*Room),
		config:   config,
		metrics:  &Metrics{},
		shutdown: make(chan struct{}),
	}
	server.loadCategories()

	distFS, err := fs.Sub(dist, "client/dist")
	if err != nil {
		log.Fatalf("Error creating sub-filesystem: %v", err)
	}
	server.distFS = distFS

	return server
}

func (s *Server) loadCategories() {
	data, err := data.ReadFile("data/categories.json")
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
		clients:        make(map[*websocket.Conn]bool),
		broadcast:      make(chan BroadcastMessage),
		register:       make(chan *websocket.Conn),
		unregister:     make(chan *websocket.Conn),
		maxClients:     maxClients,
		usedCategories: make([]string, 0),
		revealed:       0,
		lastActivity:   time.Now(),
		done:           make(chan struct{}),
	}
}

func (s *Server) getOrCreateRoom(roomID string) (*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		room = &Room{
			clients:        make(map[*websocket.Conn]bool),
			broadcast:      make(chan BroadcastMessage),
			register:       make(chan *websocket.Conn),
			unregister:     make(chan *websocket.Conn),
			maxClients:     s.config.MaxClients,
			usedCategories: make([]string, 0),
			revealed:       0,
			lastActivity:   time.Now(),
			done:           make(chan struct{}),
			server:         s,
		}
		s.rooms[roomID] = room
		s.metrics.mu.Lock()
		s.metrics.activeRooms++
		s.metrics.mu.Unlock()
		go room.run()
	}
	return room, nil
}

func (s *Server) handleWebSocket(conn *websocket.Conn, room *Room) {
	defer conn.Close()

	// Register the connection to the room
	room.register <- conn

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
			newCategory := s.getUniqueCategory(room.usedCategories)
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
			room.usedCategories = append(room.usedCategories, newCategory)
		case "reveal":
			room.revealed++
			if room.revealed == len(room.clients) {
				allRevealedMsg, err := json.Marshal(map[string]interface{}{
					"type": "allRevealed",
				})
				if err != nil {
					log.Printf("Error marshalling allRevealed message: %v", err)
					continue
				}
				room.broadcast <- BroadcastMessage{
					message: allRevealedMsg,
					sender:  conn,
					msgType: "allRevealed",
				}
				room.revealed = 0
			}
		default:
			room.broadcast <- BroadcastMessage{message: message, sender: conn, msgType: msg["type"].(string)}
		}
	}
}

func (s *Server) getUniqueCategory(usedCategories []string) string {
	if len(usedCategories) == len(s.categories) {
		usedCategories = make([]string, 0)
	}

	for {
		newCategory := s.getRandomCategory()
		if !contains(usedCategories, newCategory) {
			return newCategory
		}
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.metrics.mu.Lock()
		s.metrics.errorCount++
		s.metrics.mu.Unlock()
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		log.Println("Error: Room ID is required")
		conn.Close()
		return
	}

	room, err := s.getOrCreateRoom(roomID)
	if err != nil {
		s.metrics.mu.Lock()
		s.metrics.errorCount++
		s.metrics.mu.Unlock()
		log.Printf("Error getting or creating room: %v", err)
		conn.Close()
		return
	}

	// Check if the room is full before registering
	if len(room.clients) >= s.config.MaxClients {
		s.metrics.mu.Lock()
		s.metrics.errorCount++
		s.metrics.mu.Unlock()
		log.Printf("Room %s is full. Connection rejected.", roomID)
		conn.Close()
		return
	}

	log.Printf("New client connected to room: %s", roomID)
	s.handleWebSocket(conn, room)
}

func (r *Room) run() {
	ticker := time.NewTicker(30 * time.Second) // Heartbeat ticker
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case client := <-r.register:
			r.handleRegister(client)
		case client := <-r.unregister:
			r.handleUnregister(client)
		case broadcastMsg := <-r.broadcast:
			r.broadcastMessage(broadcastMsg)
		case <-ticker.C:
			r.sendHeartbeat()
		}
	}
}

func (r *Room) handleRegister(client *websocket.Conn) {
	if len(r.clients) < r.maxClients {
		r.clients[client] = true
		r.lastActivity = time.Now()
		r.server.metrics.mu.Lock()
		r.server.metrics.activeClients++
		r.server.metrics.mu.Unlock()
		log.Printf("Client registered. Total clients: %d", len(r.clients))
	} else {
		log.Println("Room is full. Rejecting new client.")
		client.Close()
	}
}

func (r *Room) handleUnregister(client *websocket.Conn) {
	if client == nil {
		log.Printf("Warning: Attempted to unregister nil client")
		return
	}

	if _, ok := r.clients[client]; ok {
		delete(r.clients, client)
		client.Close()
		r.lastActivity = time.Now()
		r.server.metrics.mu.Lock()
		r.server.metrics.activeClients--
		r.server.metrics.mu.Unlock()
		log.Printf("Client unregistered. Total clients: %d", len(r.clients))
	}
}

func (r *Room) broadcastMessage(broadcastMsg BroadcastMessage) {
	for client := range r.clients {
		if client == nil {
			continue
		}
		if broadcastMsg.msgType != "newCategory" && broadcastMsg.msgType != "allRevealed" && client == broadcastMsg.sender {
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

func (r *Room) sendHeartbeat() {
	heartbeat, _ := json.Marshal(map[string]interface{}{
		"type": "heartbeat",
	})

	for client := range r.clients {
		if client == nil {
			continue
		}
		err := client.WriteMessage(websocket.PingMessage, heartbeat)
		if err != nil {
			r.unregister <- client
		}
	}
}

func (s *Server) cleanupEmptyRooms() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, room := range s.rooms {
		if len(room.clients) == 0 || now.Sub(room.lastActivity) > s.config.RoomTimeout {
			close(room.broadcast)
			close(room.register)
			close(room.unregister)
			close(room.done)
			delete(s.rooms, id)
			s.metrics.mu.Lock()
			s.metrics.activeRooms--
			s.metrics.mu.Unlock()
			log.Printf("Cleaned up room: %s", id)
		}
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	close(s.shutdown)

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, room := range s.rooms {
		close(room.done)
		for client := range room.clients {
			client.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutdown"),
				time.Now().Add(time.Second),
			)
			client.Close()
		}
	}
	return nil
}

func main() {
	config := Config{
		Port:            "8080",
		MaxClients:      2,
		CleanupInterval: 5 * time.Minute,
		RoomTimeout:     30 * time.Minute,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
	}

	server := NewServer(config)

	// Setup HTTP server
	srv := &http.Server{
		Addr:         "0.0.0.0:" + config.Port,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	// Setup routes
	mux := http.NewServeMux()

	// Add basic metrics endpoint
	mux.HandleFunc("/metrics", server.handleMetrics)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Setup static file server
	fileServer := http.FileServer(http.FS(server.distFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := server.distFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	mux.HandleFunc("/ws", server.handleConnections)

	srv.Handler = mux

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				server.cleanupEmptyRooms()
			case <-server.shutdown:
				return
			}
		}
	}()

	// Start server
	go func() {
		log.Printf("Server starting on 0.0.0.0:%s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error during HTTP server shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	metrics := map[string]interface{}{
		"active_rooms":   s.metrics.activeRooms,
		"active_clients": s.metrics.activeClients,
		"messages_total": s.metrics.messagesTotal,
		"error_count":    s.metrics.errorCount,
	}

	json.NewEncoder(w).Encode(metrics)
}
