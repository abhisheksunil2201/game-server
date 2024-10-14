package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
)

type Service interface {
	Health() map[string]string
	Close() error
	StorePlayer(playerId, playerName string) error
	StoreGameHistory(gameId, players, result string) error
	UpdateGameResult(gameId, result string) error
	GetPlayerGames(playerId string) ([]map[string]interface{}, error)
}

type service struct {
	db *sql.DB
}

var (
	dburl      = os.Getenv("DB_URL")
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	db, err := sql.Open("sqlite3", dburl)
	if err != nil {
		log.Fatal(err)
	}

	dbInstance = &service{
		db: db,
	}

	createTables(db)

	return dbInstance
}

func createTables(db *sql.DB) {
	createPlayersTable := `
	CREATE TABLE IF NOT EXISTS players (
		player_id TEXT PRIMARY KEY,
		name TEXT,
		joined_at DATETIME
	);`

	createGameHistoryTable := `
	CREATE TABLE IF NOT EXISTS game_history (
		game_id TEXT PRIMARY KEY,
		players TEXT,
		start_time DATETIME,
		end_time DATETIME,
		result TEXT
	);`

	_, err := db.Exec(createPlayersTable)
	if err != nil {
		log.Fatal("Failed to create players table:", err)
	}

	_, err = db.Exec(createGameHistoryTable)
	if err != nil {
		log.Fatal("Failed to create game history table:", err)
	}
}

func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf(fmt.Sprintf("db down: %v", err))
		return stats
	}

	stats["status"] = "up"
	stats["message"] = "It's healthy"

	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	if dbStats.OpenConnections > 40 {
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

func (s *service) StorePlayer(id string, Name string) error {
	log.Printf("Creating user %s %s", id, Name)
	_, err := s.db.Exec(
		`INSERT INTO players (player_id, name, joined_at) VALUES (?, ?, datetime('now'))`,
		id, Name)
	return err
}

func (s *service) StoreGameHistory(gameID, players, result string) error {
	_, err := s.db.Exec(
		`INSERT INTO game_history (game_id, players, start_time, end_time, result) VALUES (?, ?, datetime('now'), NULL, ?)`,
		gameID, players, result)
	return err
}

func (s *service) UpdateGameResult(gameID, result string) error {
	_, err := s.db.Exec(
		`UPDATE game_history SET end_time = datetime('now'), result = ? WHERE game_id = ?`,
		result, gameID)
	return err
}

func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", dburl)
	return s.db.Close()
}

func (s *service) GetPlayerGames(playerId string) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(`SELECT game_id, start_time, end_time, result FROM game_history WHERE players LIKE '%' || ? || '%'`, playerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []map[string]interface{}
	for rows.Next() {
		var gameId, startTime, endTime, result string
		if err := rows.Scan(&gameId, &startTime, &endTime, &result); err != nil {
			return nil, err
		}
		game := map[string]interface{}{
			"gameId":    gameId,
			"startTime": startTime,
			"endTime":   endTime,
			"result":    result,
		}
		games = append(games, game)
	}

	return games, nil
}
