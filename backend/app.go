package main

import (
	a_ctrl "github.com/Secreto31126/tesis/common/controllers/auth"
	"github.com/Secreto31126/tesis/common/controllers/boards"
	g_ctrl "github.com/Secreto31126/tesis/common/controllers/groups"
	"github.com/Secreto31126/tesis/common/controllers/postits"
	s_ctrl "github.com/Secreto31126/tesis/common/controllers/secrets"
	u_ctrl "github.com/Secreto31126/tesis/common/controllers/users"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/executer"
	"github.com/Secreto31126/tesis/common/ports/jwt"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	"github.com/Secreto31126/tesis/common/ports/oauth"
	"github.com/Secreto31126/tesis/common/ports/password"
	"github.com/Secreto31126/tesis/common/ports/redis"
	a_srv "github.com/Secreto31126/tesis/common/services/auth"
	b_srv "github.com/Secreto31126/tesis/common/services/boards"
	g_srv "github.com/Secreto31126/tesis/common/services/groups"
	p_srv "github.com/Secreto31126/tesis/common/services/postits"
	r_srv "github.com/Secreto31126/tesis/common/services/realtime"
	s_srv "github.com/Secreto31126/tesis/common/services/secrets"
	u_srv "github.com/Secreto31126/tesis/common/services/users"
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

func newApp(db *mongo.MongoDB, cache *redis.RedisDB, kek *crypto.Sealer, keys *jwt.Keyring, cfg config, mailer infrastructure.Mailer) *app {
	secrets := s_srv.New(db, s_srv.NewPolicy(db, db), crypto.NewKeyring(kek, db), oauth.New(), cache, cache, db)

	authService := a_srv.New(a_srv.Config{AccessTTL: cfg.accessTTL, RefreshTTL: cfg.refreshTTL, Admins: cfg.admins},
		db, cache, password.New(), keys, cache, cache, mailer)
	userService := u_srv.New(db)

	boardService := b_srv.New(db, secrets)
	groupService := g_srv.New(db, secrets)
	postitService := p_srv.New(db, cache, executer.New(), secrets)
	realtimeService := r_srv.New(*boardService, cache)

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(cors.New(corsConfig()))

	api := router.Group("/api/v1")
	a_ctrl.NewController(authService, a_ctrl.Cookies{Secure: cfg.cookieSecure, Path: AUTH_COOKIE_PATH}).RegisterRoutes(api)
	u_ctrl.NewController(userService, keys).RegisterRoutes(api)
	boards.NewController(boardService, realtimeService).RegisterRoutes(api)
	postits.NewController(postitService).RegisterRoutes(api)
	s_ctrl.NewController(secrets).RegisterRoutes(api)
	g_ctrl.NewController(groupService).RegisterRoutes(api)

	return &app{secrets: secrets, router: router}
}
