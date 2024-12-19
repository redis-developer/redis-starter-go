package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/redis-developer/redis-starter-go/cmd/config"
	api "github.com/redis-developer/redis-starter-go/pkg"
)

func main() {
	// Load environment variables
	err := godotenv.Load()

	if err != nil {
		fmt.Println("failed to load environment file, " +
			"assuming environment variables are already loaded")
	}

	server := api.NewServer(&config.Config{})
	server.Run()
}
