package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	playerCount := 10
	var wg sync.WaitGroup
	wg.Add(playerCount)

	go func() {
		time.Sleep(3 * time.Second)
		log.Println("Closing game...")
		if gameId != "" {
			closeGame(gameId)
		} else {
			log.Println("No gameID received yet")
		}
	}()

	for i := 0; i < playerCount; i++ {
		go func(ID string) {
			defer wg.Done()
			simulatePlayer(ID)
		}(userIDs[i])
	}

	wg.Wait()
}

func simulatePlayer(ID string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws", RawQuery: fmt.Sprintf("ID=%s", ID)}
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

var userIDs = []string{
	"5fe401cf-e58b-4481-bf21-536cbdfb955b",
	"4e2b03b2-6999-44ba-b62d-5b186d4a4b43",
	"2826c4b2-5a58-440a-bb8c-0efb8716ad45",
	"ff9b8985-526d-421f-b6a5-d9397a707727",
	"3a74214-080a-4cb8-a739-01d43a921966Q",
	"dd78270d-181e-4be3-93ac-b7b34d119281",
	"ae977611-a5f9-4e6f-a5f6-b89bb621bdd3",
	"e7400550-f14c-4839-ad4f-4c2d6f503035",
	"96d8407c-db94-44dc-8fdc-cc89fe2ae4b5",
	"31ff4443-edb6-4006-a95d-476d08a64a79",
	"e34b80bd-50ff-4ef7-9202-e69ea292c389",
	"5266e40a-5c15-4271-be13-67d1248daad7",
	"f0075774-28aa-4f98-9285-9b89d1b31274",
	"5d09549d-83d0-4b9f-80a5-e5f592b6528e",
	"252c1333-31d5-4c69-885b-e8a782185dc0",
	"6bfe38a1-3ff7-4c4f-810f-37bde369aba5",
	"ecf6925e-5960-4835-99e0-9b6e81cd1c1e",
	"05e154e9-4555-4818-9ee2-39dfe1fd7007",
	"831e8f89-a78c-4895-9211-d342e44618a9",
	"669e5baa-f03d-4565-a832-0a1d2afa9cbc",
	"8687e8c8-0922-46d5-8a8b-9701be2126ba",
	"e9adbfbe-5b19-42c8-8029-1bfee194edc3",
	"870be05c-83da-4dcd-b056-b27e80b71c8a",
	"1f6fdb13-a924-4f19-b87d-83c430ae613c",
	"974f85ec-61a6-4a54-803f-20aa34754f73",
	"0a8ac8ed-7724-4bc0-8a56-118f2ebe958c",
	"b221ddfa-fc8e-4015-a9ef-c7b18fa5f56c",
	"193544c7-3d67-4a30-9869-285bf6afc051",
	"11430d89-9b57-431c-a6c5-a8ae2af71b43",
	"8b7df016-8d94-445d-b1f3-a13350c82624",
	"952bd4cf-2c1d-4af7-8d26-d8b59a37c823",
	"3ba492ea-ef29-4c11-8de5-ddb02b2df1b8",
	"bcc3ee17-0912-49eb-8645-4e35bb5cdc54",
	"7236bf01-a542-4e58-8e48-0d0b65a97137",
	"57ecd17b-c6ea-4835-b0de-9b44ed19afc8",
	"80da5bd9-58bb-4dea-bc92-5be166cfe080",
	"71abe7d6-f88b-4870-818d-f3e66fa63ef7",
	"af699945-7f9d-4407-92f8-58377e1b871f",
	"98827a34-91aa-4dcf-a23e-47d746b759de",
	"7348dafe-4118-48a8-9d72-2c0191cdea2a",
	"a7a63a1d-08cd-4cde-b807-cc77924c0729",
	"0f4fde8f-47d9-4847-aebc-900533f87841",
	"2f5339ca-9d2a-4e09-b6c6-fbdf33fa3f79",
	"1b0f3c6c-170b-4088-9620-b7b7fdac49cb",
	"00e71092-32a9-49ef-a62e-f3e7798059ef",
	"64195093-c301-427d-902d-189827c40040",
	"4c5b2271-9183-46e0-9cc1-0d6ca21b92dd",
	"29e85b77-7a14-4066-a1a6-bca6bfa93b7d",
	"6a3c09e2-efa1-44ba-966e-ed8c7671e223",
	"26a0e565-4e91-4445-bace-89108c43539e",
}
