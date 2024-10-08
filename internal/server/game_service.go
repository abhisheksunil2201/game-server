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
	ID         string
	LastActive time.Time
	Name       string
}

type Game struct {
	ID        string
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

func (s *Server) PlayerConnect(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection: ", err)
		return
	}
	player := &Player{Conn: ws, ID: uuid.New().String()}
	playerQueue <- player
	log.Printf("Player %s connected", player.ID)

	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				break
			}
			mu.Lock()
			player.LastActive = time.Now()
			mu.Unlock()
		}
	}()
}

func Matchmaking() {
	log.Println("********* Matchmaking active *********")

	for {
		if len(playerQueue) >= 6 {
			players := make([]*Player, 6)
			for i := 0; i < 6; i++ {
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
		ID:       gameId,
		Players:  players,
		Ticker:   ticker,
		StopChan: stopChan,
	}

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
			log.Printf("Error sending game Id to player %s: %v", player.ID, err)
			player.Conn.Close()
		}
	}

	go checkPlayerInactivity(players, game.StopChan)
	go gameTickerLoop(game, ticker, stopChan)
}

func checkPlayerInactivity(players []*Player, stopChan chan struct{}) {
	inactivityTimeout := 30 * time.Second
	for {
		select {
		case <-time.After(10 * time.Second):
			mu.Lock()
			for _, player := range players {
				if time.Since(player.LastActive) > inactivityTimeout {
					log.Printf("Player %s is inactive, disconnecting...", player.ID)
					player.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Disconnected due to inactivity"))
					player.Conn.Close()
				}
			}
			mu.Unlock()
		case <-stopChan:
			return
		}
	}
}

func gameTickerLoop(game *Game, ticker *time.Ticker, stopChan chan struct{}) {
	for {
		select {
		case <-ticker.C:
			game.GameState = getGameState(game.ID)
			for _, player := range game.Players {
				err := player.Conn.WriteJSON(game.GameState)
				if err != nil {
					player.Conn.Close()
				}
			}
		case <-stopChan:
			log.Printf("Closing game %s", game.ID)
			for _, player := range game.Players {
				player.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Game over"))
				player.Conn.Close()
			}
			ticker.Stop()
			return
		}
	}
}

func CloseGameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameId := vars["gameId"]

	mu.Lock()
	_, exists := activeGames[gameId]
	mu.Unlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	CloseGame(gameId)

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

func (s *Server) CreatePlayerHandler(w http.ResponseWriter, r *http.Request) {
	var player Player
	err := json.NewDecoder(r.Body).Decode(&player)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.db.StorePlayer(player.ID, player.Name)
	response := map[string]string{
		"message":  "PLayer created successfully",
		"playerId": player.ID,
	}
	jsonResponse(w, response, http.StatusCreated)

}
