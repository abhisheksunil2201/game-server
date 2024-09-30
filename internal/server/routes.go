package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Player struct {
	Conn *websocket.Conn
	id   string
}

type Game struct {
	id        string
	Players   []*Player
	Ticker    *time.Ticker
	StopChan  chan struct{}
	GameState string
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

	go func() {
		for {
			select {
			case <-ticker.C:
				game.GameState = getGameState(gameId)
				for _, player := range players {
					err := player.Conn.WriteJSON(game.GameState)
					if err != nil {
						log.Printf("Error sending message to player %s: %v", player.id, err)
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

func getGameState(gameId string) string {
	return fmt.Sprintf("Game state %s", gameId)
}
