package main

import (
	"context"

	"github.com/spyder01/lilium-go"
)

func main() {
	cfg := lilium.LoadConfig("test.yaml")
	app := lilium.New(cfg, context.Background())
	router := lilium.NewRouter(app.Context)

	app.Start(router)
}
