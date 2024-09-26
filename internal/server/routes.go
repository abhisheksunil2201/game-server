package server

import (
	"log"
	"net/http"
  "encoding/json"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
  CheckOrigin: func(r *http.Request) bool {
    return true
  },
}

func (s *Server) RegisterRoutes() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", s.helloHandler)
	r.HandleFunc("/ws", s.websocketHandler)

	return r
}

func (s *Server) helloHandler(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("Hello from the server"))
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request){
    log.Println("Attempting to upgrade to WebSocket...")
    
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
      log.Println(err)
      return
    }

    log.Println("WebSocket upgrade successful")

    s.handleConnections(ws)
  
}

func (s *Server) handleConnections(ws *websocket.Conn) {
  s.mutex.Lock()
  s.clients[ws] = true
  s.mutex.Unlock()

  log.Println("New client connected")

  disconnected := make(chan struct{})
  go s.readPump(ws, disconnected)
  <-disconnected
  s.mutex.Lock()
  delete(s.clients, ws)
  s.mutex.Unlock()

  log.Println("Client disconnected")
}

func (s *Server) readPump(ws *websocket.Conn, disconnected chan struct{}) {
  defer func () {
    ws.Close()
    close(disconnected)
  }()

  for {
		var msg map[string]interface{}
		err := ws.ReadJSON(&msg)
		if err != nil {
      log.Printf("error on ReadJSON: %v", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

    jsonData, err := json.MarshalIndent(msg, "", "  ")
    if err != nil {
      log.Printf("error formatting JSON: %v", err)
    } else {
      log.Printf("Received JSON message: %s", string(jsonData))
    }

		err = ws.WriteJSON(msg)
		if err != nil {
			log.Println("error writing message:", err)
			break
		}
	}
}
