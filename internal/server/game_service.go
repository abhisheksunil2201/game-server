package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
	userId := r.URL.Query().Get("ID")
	lastActiveStr := r.URL.Query().Get("LastActive")
	name := r.URL.Query().Get("Name")
	if userId == "" {
		http.Error(w, "Missing userId", http.StatusBadRequest)
		return
	}
	if name == "" {
		http.Error(w, "Missing name", http.StatusBadRequest)
		return
	}
	if lastActiveStr == "" {
		http.Error(w, "Missing lastActive", http.StatusBadRequest)
		return
	}
	lastActive, err := time.Parse(time.RFC3339, lastActiveStr)
	if err != nil {
		log.Printf("Error parsing LastActive: %v", err)
		http.Error(w, "Invalid LastActive timestamp", http.StatusBadRequest)
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection: ", err)
		return
	}
	player := &Player{Conn: ws, ID: userId, Name: name, LastActive: lastActive}
	playerQueue <- player
	log.Printf("Player %s %s connected", player.ID, player.Name)

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

func (s *Server) Matchmaking() {
	log.Println("********* Matchmaking active *********")

	for {
		if len(playerQueue) >= 6 {
			players := make([]*Player, 6)
			for i := 0; i < 6; i++ {
				players[i] = <-playerQueue
			}
			go s.StartMatch(players)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Server) StartMatch(players []*Player) {
	gameId := uuid.New().String()
	stopChan := make(chan struct{})
	ticker := time.NewTicker(16 * time.Millisecond)
	playerNames := []string{}
	for _, player := range players {
		playerNames = append(playerNames, player.Name)
	}
	// Add game to the game_history table when the game starts
	playersStr := strings.Join(playerNames, ",")
	err := s.db.StoreGameHistory(gameId, playersStr, "in-progress")
	if err != nil {
		log.Printf("Error storing game history: %v", err)
	}
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
	go gameTickerLoop(game, ticker, game.StopChan)
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

func (s *Server) CloseGameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameId := vars["gameId"]

	mu.Lock()
	_, exists := activeGames[gameId]
	mu.Unlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	s.CloseGame(gameId)

	response := map[string]string{
		"message": "Game closed successfully",
		"gameId":  gameId,
	}
	jsonResponse(w, response, http.StatusOK)
}

func (s *Server) CloseGame(gameId string) {
	mu.Lock()
	game, exists := activeGames[gameId]

	if exists {
		for _, player := range game.Players {
			player.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Game over"))
			player.Conn.Close()
		}
		close(game.StopChan)
		// Update game result and end time in the game_history table
		err := s.db.UpdateGameResult(gameId, "finished")
		if err != nil {
			log.Printf("Error updating game result: %v", err)
		}
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

func (s *Server) GetPlayerGameHistory(w http.ResponseWriter, r *http.Request) {
	playerId := r.URL.Query().Get("playerId")
	if playerId == "" {
		http.Error(w, "Missing playerId", http.StatusBadRequest)
		return
	}

	games, err := s.db.GetPlayerGames(playerId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, games, http.StatusOK)
}
