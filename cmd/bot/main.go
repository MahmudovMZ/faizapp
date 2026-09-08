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

	//Loading config files
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	//building a dsn string
	dsn := database.BuildDSN(cfg.DB)
	err = database.RunMigrations(dsn)
	if err != nil {
		log.Fatal("Error loading database migrations: ", err)
	}

	//creating a pool of connection
	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}
	defer pool.Close()

	//running the application
	err = app.Run(*cfg, pool)
	if err != nil {
		log.Println("Error starting app: ", err)
		return
	}
}
