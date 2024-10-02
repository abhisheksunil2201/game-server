package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Player struct {
	Conn       *websocket.Conn
	id         string
	LastActive time.Time
}

type Game struct {
	id        string
	Players   []*Player
	Ticker    *time.Ticker
	StopChan  chan struct{}
	GameState map[string]interface{}
}

var playerQueue = make(chan *Player, 100)
var activeGames = make(map[string]*Game)
var mu sync.Mutex

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
	r.HandleFunc("/ws", s.PlayerConnect)
	r.HandleFunc("/close-game/{gameId}", CloseGameHandler).Methods("POST")

	go Matchmaking()

	return r
}

func (s *Server) helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from the server"))
}

func (s *Server) PlayerConnect(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection: ", err)
		return
	}
	player := &Player{Conn: ws, id: uuid.New().String()}
	playerQueue <- player
	log.Printf("Player %s connected", player.id)

	//For player inactivity Check
	go func() {
		for {
			// Each time the player sends a message, update LastActive
			_, _, err := ws.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				break
			}
			mu.Lock()
			player.LastActive = time.Now() // Update activity timestamp
			mu.Unlock()
		}
	}()
}

func Matchmaking() {
	log.Println("********* Matchmaking active *********")

	for {
		if len(playerQueue) >= 10 {
			players := make([]*Player, 10)
			for i := 0; i < 10; i++ {
				players[i] = <-playerQueue
			}
			go StartMatch(players)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func StartMatch(players []*Player) {
	gameId := uuid.New().String()
	stopChan := make(chan struct{})
	ticker := time.NewTicker(16 * time.Millisecond)

	game := &Game{
		id:       gameId,
		Players:  players,
		Ticker:   ticker,
		StopChan: stopChan,
	}

	// Store active game
	mu.Lock()
	activeGames[gameId] = game
	mu.Unlock()

	log.Printf("Starting game %s with players: %v\n", gameId, players)

	for _, player := range players {
		err := player.Conn.WriteJSON(map[string]string{
			"gameId":  gameId,
			"message": "Game has started",
		})
		if err != nil {
			log.Printf("Error sending game Id to player %s: %v", player.id, err)
			player.Conn.Close()
		}
	}

	// Periodically check for player inactivity
	go func() {
		inactivityTimeout := 30 * time.Second
		for {
			select {
			case <-time.After(10 * time.Second): // Check every 10 seconds
				mu.Lock()
				for _, player := range players {
					if time.Since(player.LastActive) > inactivityTimeout {
						log.Printf("Player %s is inactive, disconnecting...", player.id)
						player.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Disconnected due to inactivity"))
						player.Conn.Close()
					}
				}
				mu.Unlock()
			case <-game.StopChan:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ticker.C:
				game.GameState = getGameState(gameId)
				for _, player := range players {
					err := player.Conn.WriteJSON(game.GameState)
					if err != nil {
						// log.Printf("Error sending message to player %s: %v", player.id, err)
						player.Conn.Close()
					}
				}
			case <-stopChan:
				// Game close logic
				log.Printf("Closing game %s", gameId)
				for _, player := range players {
					player.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Game over"))
					player.Conn.Close()
				}
				ticker.Stop()
				return
			}
		}
	}()
}

func CloseGameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameId := vars["gameId"]
	for id, game := range activeGames {
		log.Printf("GameId: %s, State: %s", id, game.GameState)
	}

	mu.Lock()
	_, exists := activeGames[gameId]
	mu.Unlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Close the game
	CloseGame(gameId)

	// Send response
	response := map[string]string{
		"message": "Game closed successfully",
		"gameId":  gameId,
	}
	jsonResponse(w, response, http.StatusOK)
}

func CloseGame(gameId string) {
	mu.Lock()
	game, exists := activeGames[gameId]

	for _, player := range game.Players {
		player.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Game over"))
		player.Conn.Close()
	}

	if exists {
		close(game.StopChan)
		delete(activeGames, gameId)
		log.Printf("Game %s has been closed", gameId)
	} else {
		log.Printf("Game not found here")
	}
	mu.Unlock()
}

func getGameState(gameId string) map[string]interface{} {
	return map[string]interface{}{
		"gameId":  gameId,
		"state":   "active",
		"tick":    time.Now().Unix(),
		"message": "Game state update",
	}
}

func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
