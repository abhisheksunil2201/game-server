# Game Server

A multiplayer game server that can be configured for any multiplayer game and changes be made accordingly.

Built using Go and Gorilla WebSocket(A Go implementation of the WebSocket protocol).

## Features

- Real-time Websocket Communication: Utilizes gorilla/websocket for efficient, bidirectional communication between server and clients.
- Robust Matchmaking System: Automatically groups players into games of 6(default - can be configured).
- Scalable Architecture: Designed to handle multiple concurrent games and players.
- Player Inactivity Detection: Automatically disconnects inactive players and updates the game state.
- Game State Management: Efficiently manages and updates game states for all active games.
- Database Integration: Uses SQLite for persistent storage of player data and game history.
- RESTful API Endpoints: Provides endpoints for player creation, game closure, and retrieving player game history.
- Graceful Game Closure: Implements a mechanism to safely close games and disconnect players.
- Health Check System: Includes a comprehensive health check for monitoring database connections and server status.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

A client simulation is provided for testing the server's capabilities:
Simulates multiple concurrent player connections.
Automatically creates and closes games.
Sends periodic player actions to test real-time communication.

- Ensure Go is installed on your system.
- Clone the repository.

## MakeFile

build the application

```bash
make build
```

run the application(server)

```bash
make run
```

run the client(to test the server)

```bash
make run-client
```

## Future enhancements

- Implement load balancing for distributing player connections across multiple server instances.
- Add more sophisticated matchmaking algorithms based on player skill levels.
- Integrate with a message queue system for better scaling of game events processing.
- Implement real-time analytics for monitoring server performance and player engagement.
