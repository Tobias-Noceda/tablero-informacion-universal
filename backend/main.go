package main

import (
	"github.com/Secreto31126/tesis/common/controllers/boards"
	"github.com/Secreto31126/tesis/common/controllers/postits"
	s_ctrl "github.com/Secreto31126/tesis/common/controllers/secrets"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/executer"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	"github.com/Secreto31126/tesis/common/ports/oauth"
	"github.com/Secreto31126/tesis/common/ports/redis"
	b_srv "github.com/Secreto31126/tesis/common/services/boards"
	p_srv "github.com/Secreto31126/tesis/common/services/postits"
	r_srv "github.com/Secreto31126/tesis/common/services/realtime"
	s_srv "github.com/Secreto31126/tesis/common/services/secrets"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	ADDR = "0.0.0.0:31126"
)

func corsConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	return config
}

func main() {
	db, err := mongo.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	cache, err := redis.New()
	if err != nil {
		panic(err)
	}
	defer cache.Close()

	sealer, err := crypto.New()
	if err != nil {
		panic(err)
	}

	executer := executer.New()
	secrets := s_srv.New(db, db, sealer, oauth.New(), cache, cache)

	boardService := b_srv.New(db)
	postitService := p_srv.New(db, cache, executer, secrets)
	realtimeService := r_srv.New(*boardService, cache)

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(cors.New(corsConfig()))

	boardController := boards.NewController(boardService, realtimeService)
	postitController := postits.NewController(postitService)
	secretController := s_ctrl.NewController(secrets)

	api := router.Group("/v1")
	boardController.RegisterRoutes(api)
	postitController.RegisterRoutes(api)
	secretController.RegisterRoutes(api)

	router.Run(ADDR)
}
