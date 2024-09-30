package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	playerCount := 10
	var wg sync.WaitGroup
	wg.Add(playerCount)

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
				log.Printf("Read:", err)
				return
			}
			log.Printf("Received: %s", message)
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
				log.Println("Write:", err)
				return
			}
			log.Printf("Sent: %v", msg)

		case <-interrupt:
			log.Printf("Interrupt received, closing connection.")

			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("CloseL", err)
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
