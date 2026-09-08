package main

import (
	"context"
	"log"
	"time"

	database "github.com/MahmudovMZ/faizapp/internal/Database"
	"github.com/MahmudovMZ/faizapp/internal/app"
	"github.com/MahmudovMZ/faizapp/internal/config"
)

func main() {
	ctx, timeout := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeout()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}
	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}
	defer pool.Close()
	app.Run(*cfg, pool)
}
