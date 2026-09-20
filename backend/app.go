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

// app is the wired service graph. Built once here so the server and the
// end-to-end tests run the exact same composition.
type app struct {
	secrets *s_srv.SecretsService
	router  *gin.Engine
}

func corsConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	return config
}

func newApp(db *mongo.MongoDB, cache *redis.RedisDB, kek *crypto.Sealer) *app {
	secrets := s_srv.New(db, s_srv.NewPolicy(db), crypto.NewKeyring(kek, db), oauth.New(), cache, cache)

	boardService := b_srv.New(db, secrets)
	postitService := p_srv.New(db, cache, executer.New(), secrets)
	realtimeService := r_srv.New(*boardService, cache)

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(cors.New(corsConfig()))

	api := router.Group("/api/v1")
	boards.NewController(boardService, realtimeService).RegisterRoutes(api)
	postits.NewController(postitService).RegisterRoutes(api)
	s_ctrl.NewController(secrets).RegisterRoutes(api)

	return &app{secrets: secrets, router: router}
}
