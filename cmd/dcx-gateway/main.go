package main

import (
	"log"
	"os"

	"dcx-gateway/internal/bootstrap"
)

func main() {
	os.Exit(run())
}

func run() int {
	app, err := bootstrap.NewApp("configs/config.dev.yaml")
	if err != nil {
		log.Printf("failed to init app: %v", err)
		return 1
	}
	defer app.Close()

	if err := app.Run(); err != nil {
		log.Printf("failed to run app: %v", err)
		return 1
	}

	return 0
}
