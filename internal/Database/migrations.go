package Database

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(dbURL string) error {
	log.Println("[DATABASE] running migrations")

	m, err := migrate.New("file://migrations", dbURL)

	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("[DATABASE] database already up to date")
			return nil
		}

		return err

	}

	log.Println("[DATABASE] database migrations applied successfully")
	return nil
}
