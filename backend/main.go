package main

import (
	"github.com/Secreto31126/tesis/common/controllers/boards"
	"github.com/Secreto31126/tesis/common/controllers/postits"
	s_ctrl "github.com/Secreto31126/tesis/common/controllers/secrets"
	"github.com/Secreto31126/tesis/common/middleware"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/executer"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	"github.com/Secreto31126/tesis/common/ports/redis"
	b_srv "github.com/Secreto31126/tesis/common/services/boards"
	p_srv "github.com/Secreto31126/tesis/common/services/postits"
	s_srv "github.com/Secreto31126/tesis/common/services/secrets"
	"github.com/gin-gonic/gin"
)

const (
	ADDR = "0.0.0.0:31126"
)

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
	secrets := s_srv.New(db, db, sealer)

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(middleware.CORSMiddleware())

	boardController := boards.NewController(b_srv.New(db))
	postitController := postits.NewController(p_srv.New(db, cache, executer, secrets))
	secretController := s_ctrl.NewController(secrets)

	api := router.Group("/v1")
	boardController.RegisterRoutes(api)
	postitController.RegisterRoutes(api)
	secretController.RegisterRoutes(api)

	router.Run(ADDR)
}
