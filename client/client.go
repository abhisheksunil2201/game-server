package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID         string
	Name       string
	LastActive time.Time
}

var gameId string

func main() {
	playerCount := 10
	var wg sync.WaitGroup
	wg.Add(playerCount)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(3 * time.Second)
		log.Println("Closing game...")
		if gameId != "" {
			closeGame(gameId, cancel)
			closeAllPLayerWS(cancel)
		} else {
			log.Println("No gameID received yet")
		}
	}()

	for _, player := range playersFromDB {
		go func(ID string, Name string, LastActive time.Time) {
			defer wg.Done()
			simulatePlayer(ID, Name, LastActive, ctx)
		}(player.ID, player.Name, player.LastActive)
	}

	wg.Wait()
}

func simulatePlayer(ID string, Name string, LastActive time.Time, ctx context.Context) {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws", RawQuery: url.Values{
		"ID":         {ID},
		"LastActive": {LastActive.Format(time.RFC3339)},
		"Name":       {Name},
	}.Encode()}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Dial:", err)
	}
	defer conn.Close()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("Player %s: Game is closing, disconnecting...", ID)
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			default:
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("Player %s Read error: %v", ID, err)
					return
				}
				if messageType == websocket.CloseMessage {
					log.Printf("Player %s: Received close message from server", ID)
					return
				}
				var msg map[string]interface{}
				if err := json.Unmarshal(message, &msg); err != nil {
					log.Printf("Error decoding JSON: %v", err)
					continue
				}
				if id, ok := msg["gameId"].(string); ok {
					gameId = id
					log.Printf("Received gameId: %s", gameId)
				}
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Player %s: game stopped", ID)
			ticker.Stop()
			return
		case t := <-ticker.C:
			msg := map[string]string{
				"timestamp": t.String(),
				"action":    "move",
				"direction": "north",
			}

			err := conn.WriteJSON(msg)
			if err != nil {
				log.Printf("Player %s Write Error: %s", ID, err)
				return
			}
			log.Printf("Sent: %v", msg)
		}
	}
}

func closeAllPLayerWS(cancel context.CancelFunc) {
	log.Println("Closing all players' WS connectins")
	cancel()
}

func closeGame(gameID string, cancel context.CancelFunc) {
	apiURL := "http://localhost:8080/close-game/" + gameID
	requestBody, err := json.Marshal(map[string]string{
		"gameID": gameID,
	})

	if err != nil {
		log.Printf("Error creating JSON request: %v", err)
		return
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		log.Printf("Error creating POST request: %v", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request to close game: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("Successfully closed game %s", gameID)
	} else {
		log.Printf("Failed to close game %s: Status %d", gameID, resp.StatusCode)
	}
	cancel()
}

var playersFromDB = []Player{
	{ID: "5fe401cf-e58b-4481-bf21-536cbdfb955b", Name: "Chris Miller", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "4e2b03b2-6999-44ba-b62d-5b186d4a4b43", Name: "Chris Johnson", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "2826c4b2-5a58-440a-bb8c-0efb8716ad45", Name: "Jane Jones", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "ff9b8985-526d-421f-b6a5-d9397a707727", Name: "David Johnson", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "33a74214-080a-4cb8-a739-01d43a921966", Name: "John Miller", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "dd78270d-181e-4be3-93ac-b7b34d119281", Name: "David Jones", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "ae977611-a5f9-4e6f-a5f6-b89bb621bdd3", Name: "Emily Williams", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "e7400550-f14c-4839-ad4f-4c2d6f503035", Name: "Jane Johnson", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "96d8407c-db94-44dc-8fdc-cc89fe2ae4b5", Name: "Emma Williams", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "31ff4443-edb6-4006-a95d-476d08a64a79", Name: "David Davis", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "e34b80bd-50ff-4ef7-9202-e69ea292c389", Name: "Emily Williams", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "5266e40a-5c15-4271-be13-67d1248daad7", Name: "David Williams", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "f0075774-28aa-4f98-9285-9b89d1b31274", Name: "Jane Smith", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "5d09549d-83d0-4b9f-80a5-e5f592b6528e", Name: "Mike Taylor", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "252c1333-31d5-4c69-885b-e8a782185dc0", Name: "Emily Taylor", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "6bfe38a1-3ff7-4c4f-810f-37bde369aba5", Name: "David Johnson", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "ecf6925e-5960-4835-99e0-9b6e81cd1c1e", Name: "Chris Taylor", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "05e154e9-4555-4818-9ee2-39dfe1fd7007", Name: "Katie Taylor", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "831e8f89-a78c-4895-9211-d342e44618a9", Name: "John Williams", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
	{ID: "669e5baa-f03d-4565-a832-0a1d2afa9cbc", Name: "Jane Taylor", LastActive: time.Date(2024, 10, 8, 16, 28, 3, 0, time.UTC)},
}
