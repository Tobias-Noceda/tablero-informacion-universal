package main

import (
	"log"
	"os"

	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/jwt"
	"github.com/Secreto31126/tesis/common/ports/mail"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	"github.com/Secreto31126/tesis/common/ports/redis"
)

const (
	PORT = "31126"
)

func main() {
	db, err := mongo.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.EnsureIndexes(); err != nil {
		panic(err)
	}

	cache, err := redis.New()
	if err != nil {
		panic(err)
	}
	defer cache.Close()

	kek, err := crypto.New()
	if err != nil {
		panic(err)
	}

	keys, err := jwt.New()
	if err != nil {
		panic(err)
	}

	cfg, err := loadConfig()
	if err != nil {
		panic(err)
	}

	application := newApp(db, cache, kek, keys, cfg, mail.Log{})

	maintenance, err := parseMaintenance(os.Args[1:])
	if err != nil {
		panic(err)
	}
	if maintenance.requested() {
		if err := maintenance.run(application.secrets, os.Stdout); err != nil {
			panic(err)
		}
		return
	}

	if missing, err := application.secrets.MissingSystemSecrets(); err != nil {
		log.Printf("could not check system secrets: %v", err)
	} else if len(missing) > 0 {
		log.Printf("system secrets not configured, well-knowns needing them will fail: %v", missing)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = PORT
	}

	application.router.Run(":" + port)
}
