package server

import (
	"encoding/json"
	"log"
	"net/http"
	"fmt"
	"time"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func (s *Server) RegisterRoutes() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/websocket", s.websocketHandler)

	return r
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request){} 
