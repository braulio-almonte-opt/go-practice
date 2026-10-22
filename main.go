package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/braulio-almonte-opt/go-practice/models"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	client, err := models.ConnectDatabase()
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("disconnect MongoDB: %v", err)
		}
	}()

	r := initializeRouter(client)
	log.Println("Server started")
	if err := r.Run(); err != nil {
		return err
	}
	return nil
}
