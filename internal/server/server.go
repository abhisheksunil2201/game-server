package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
  "sync"
  "github.com/gorilla/websocket"

	_ "github.com/joho/godotenv/autoload"

	"game-server/internal/database"
)

type Server struct {
	port int
  clients map[*websocket.Conn]bool
	mutex sync.Mutex 
  db database.Service
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port: port,
    clients: make(map[*websocket.Conn]bool),
		db: database.New(),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
