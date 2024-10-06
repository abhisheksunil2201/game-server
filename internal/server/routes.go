package server

import (
	"github.com/gorilla/mux"
	"net/http"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", s.helloHandler)
	r.HandleFunc("/ws", s.PlayerConnect)
	r.HandleFunc("/close-game/{gameId}", CloseGameHandler).Methods("POST")

	go Matchmaking()

	return r
}

func (s *Server) helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from the server"))
}
