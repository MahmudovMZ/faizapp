package main

import (
	"log"

	"github.com/MahmudovMZ/faizapp/internal/app"
	"github.com/MahmudovMZ/faizapp/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	app.Run(*cfg)
}
