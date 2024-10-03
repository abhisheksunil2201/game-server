package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var gameId string

func main() {
	playerCount := 10000
	var wg sync.WaitGroup
	wg.Add(playerCount)

	// go func() {
	// 	time.Sleep(3 * time.Second)
	// 	log.Println("Closing game...")
	// 	if gameId != "" {
	// 		closeGame(gameId)
	// 	} else {
	// 		log.Println("No gameID received yet")
	// 	}
	// }()

	for i := 0; i < playerCount; i++ {
		go func() {
			defer wg.Done()
			simulatePlayer()
		}()
	}

	wg.Wait()
}

func simulatePlayer() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Dial:", err)
	}
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Read: %s", err)
				return
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Error decoding JSON: %v", err)
				log.Printf("Received raw message: %s", string(message))
				continue
			}

			if id, ok := msg["gameId"].(string); ok {
				gameId = id
				log.Printf("Received gameId: %s", gameId)

			} //else {
			// log.Printf("Received: %s", message)
			//}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			msg := map[string]string{
				"timestamp": t.String(),
				"action":    "move",
				"direction": "north",
			}

			err := conn.WriteJSON(msg)
			if err != nil {
				log.Printf("Write: %s", err)
				return
			}
			log.Printf("Sent: %v", msg)

		case <-interrupt:
			log.Printf("Interrupt received, closing connection.")

			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("Close: %s", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func closeGame(gameID string) {
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
}
