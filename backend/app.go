package main

import (
	"time"

	a_ctrl "github.com/Secreto31126/tesis/common/controllers/auth"
	bl_ctrl "github.com/Secreto31126/tesis/common/controllers/billing"
	"github.com/Secreto31126/tesis/common/controllers/boards"
	g_ctrl "github.com/Secreto31126/tesis/common/controllers/groups"
	"github.com/Secreto31126/tesis/common/controllers/middleware"
	o_ctrl "github.com/Secreto31126/tesis/common/controllers/orgs"
	"github.com/Secreto31126/tesis/common/controllers/postits"
	cw_ctrl "github.com/Secreto31126/tesis/common/controllers/postittemplates"
	s_ctrl "github.com/Secreto31126/tesis/common/controllers/secrets"
	t_ctrl "github.com/Secreto31126/tesis/common/controllers/tokens"
	u_ctrl "github.com/Secreto31126/tesis/common/controllers/users"
	w_ctrl "github.com/Secreto31126/tesis/common/controllers/wellknowns"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/executer"
	"github.com/Secreto31126/tesis/common/ports/jwt"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	"github.com/Secreto31126/tesis/common/ports/oauth"
	"github.com/Secreto31126/tesis/common/ports/password"
	"github.com/Secreto31126/tesis/common/ports/redis"
	"github.com/Secreto31126/tesis/common/services/access"
	a_srv "github.com/Secreto31126/tesis/common/services/auth"
	bl_srv "github.com/Secreto31126/tesis/common/services/billing"
	b_srv "github.com/Secreto31126/tesis/common/services/boards"
	e_srv "github.com/Secreto31126/tesis/common/services/entitlements"
	g_srv "github.com/Secreto31126/tesis/common/services/groups"
	o_srv "github.com/Secreto31126/tesis/common/services/orgs"
	p_srv "github.com/Secreto31126/tesis/common/services/postits"
	cw_srv "github.com/Secreto31126/tesis/common/services/postittemplates"
	s_srv "github.com/Secreto31126/tesis/common/services/secrets"
	t_srv "github.com/Secreto31126/tesis/common/services/tokens"
	u_srv "github.com/Secreto31126/tesis/common/services/users"
	w_srv "github.com/Secreto31126/tesis/common/services/wellknowns"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// app is the wired service graph. Built once here so the server and the
// end-to-end tests run the exact same composition.
type app struct {
	secrets *s_srv.SecretsService
	billing *bl_srv.Service
	router  *gin.Engine
}

func corsConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	return config
}

func newApp(db *mongo.MongoDB, cache *redis.RedisDB, kek *crypto.Sealer, keys *jwt.Keyring, cfg config, mailer infrastructure.Mailer) *app {
	resolver := access.New(db)
	// What each user's plan allows; every service that creates something
	// asks it who pays and how much room they have left.
	plans := e_srv.New(db, db, db, db, db, time.Now)
	secrets := s_srv.New(db, s_srv.NewPolicy(db, db, resolver), crypto.NewKeyring(kek, db), oauth.New(), cache, cache, db, resolver, plans)

	authService := a_srv.New(a_srv.Config{AccessTTL: cfg.accessTTL, RefreshTTL: cfg.refreshTTL, Admins: cfg.admins},
		db, cache, password.New(), keys, cache, cache, mailer, cfg.identityProvider())
	userService := u_srv.New(db)

	boardService := b_srv.New(db, secrets, resolver, db, cache, db, plans)
	groupService := g_srv.New(db, secrets, resolver, plans)
	orgService := o_srv.New(db, db, secrets, cache, plans)
	postitTemplateService := cw_srv.New(db, db, resolver, plans)
	postitService := p_srv.New(db, cache, executer.New(), secrets, resolver, db, postitTemplateService, plans, cache)
	wellKnownService := w_srv.New(db)
	tokenService := t_srv.New(db, cache, plans, time.Now)

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(cors.New(corsConfig()))

	api := router.Group("/api/v1")
	a_ctrl.NewController(authService, a_ctrl.Cookies{Secure: cfg.cookieSecure, Path: AUTH_COOKIE_PATH}).RegisterRoutes(api)

	// The plans on offer are public (the pricing page shows them signed
	// out); what the caller has used of theirs is not.
	gateway := cfg.paymentGateway()
	billingService := bl_srv.New(gateway, db, db, bl_srv.Config{USDARS: cfg.billingUSDARS}, time.Now)
	billing := bl_ctrl.NewController(plans, billingService, gateway)
	billing.RegisterPublicRoutes(api)

	// Everything but /auth needs a signed-in user; handlers read who it is
	// from middleware.Principal. A personal access token signs in too, but
	// only reaches the read routes RestrictAccessTokens allows.
	private := api.Group("", middleware.RequireAuth(tokenService.Verifier(keys)), middleware.RestrictAccessTokens())
	u_ctrl.NewController(userService).RegisterRoutes(private)
	t_ctrl.NewController(tokenService).RegisterRoutes(private)
	boards.NewController(boardService).RegisterRoutes(private)
	postits.NewController(postitService).RegisterRoutes(private)
	s_ctrl.NewController(secrets).RegisterRoutes(private)
	g_ctrl.NewController(groupService).RegisterRoutes(private)
	o_ctrl.NewController(orgService).RegisterRoutes(private)
	w_ctrl.NewController(wellKnownService).RegisterRoutes(private)
	cw_ctrl.NewController(postitTemplateService).RegisterRoutes(private)
	billing.RegisterRoutes(private)

	return &app{secrets: secrets, billing: billingService, router: router}
}
