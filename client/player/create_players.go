package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"net/http"
)

type Player struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	firstNames = []string{"John", "Jane", "Alex", "Emily", "Chris", "Katie", "Mike", "Laura", "David", "Emma"}
	lastNames  = []string{"Smith", "Johnson", "Williams", "Jones", "Brown", "Davis", "Miller", "Wilson", "Moore", "Taylor"}
)

// Generates a random full name
func generateRandomName() string {
	firstName := firstNames[rand.Intn(len(firstNames))]
	lastName := lastNames[rand.Intn(len(lastNames))]
	return fmt.Sprintf("%s %s", firstName, lastName)
}

func main() {
	for i := 0; i < 100; i++ {
		playerId := uuid.New().String()
		playerName := generateRandomName()
		player := Player{
			ID:   playerId,
			Name: playerName,
		}

		// Convert the player struct to JSON
		playerJSON, err := json.Marshal(player)
		if err != nil {
			fmt.Printf("Error marshalling player: %v\n", err)
			continue
		}

		// Send POST request to the endpoint
		req, err := http.NewRequest("POST", "http://localhost:8080/create-player", bytes.NewBuffer(playerJSON))
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			fmt.Printf("Error sending request: %v\n", err)
			continue
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error sending request to close game: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusCreated {
			fmt.Printf("Successfully created player: %s\n", playerName)
		} else {
			fmt.Printf("Failed to create player. Status Code: %d\n", resp.StatusCode)
		}
	}
}
