package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"wcstransfer/backend/internal/app"
	"wcstransfer/backend/internal/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded: %v", err)
	}

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := app.NewServer(cfg)
	if err := server.Run(ctx); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
