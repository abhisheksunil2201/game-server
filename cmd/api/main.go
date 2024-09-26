package main

import (
	"fmt"
	"game-server/internal/server"
  "log"

)

func main() {
	server := server.NewServer()
  log.Println("Starting server")
	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}
}
