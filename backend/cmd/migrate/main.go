package main

import (
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog/log"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal().Msg("DATABASE_URL required")
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("init migrate")
	}

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal().Err(err).Msg("migrate up")
		}
		fmt.Println("migrations applied")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal().Err(err).Msg("migrate down")
		}
		fmt.Println("migration reverted")
	default:
		fmt.Println("usage: migrate [up|down]")
	}
}
