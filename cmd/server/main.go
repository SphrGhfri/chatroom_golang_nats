package main

import (
	"flag"
	"os"

	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/app"
)

var configPath = flag.String("config", "config.json", "service configuration file")

func main() {
	flag.Parse()
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		*configPath = envPath
	}

	cfg := config.MustReadConfig(*configPath)

	application, err := app.NewApp(cfg)
	if err != nil {
		panic(err)
	}

	// Block until Stop is called or an error occurs
	if err := application.Start(); err != nil {
		panic(err)
	}
}
