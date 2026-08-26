package main

import (
	"github.com/Secreto31126/tesis/common/controllers/boards"
	"github.com/Secreto31126/tesis/common/controllers/postits"
	"github.com/Secreto31126/tesis/common/ports/executer"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	"github.com/Secreto31126/tesis/common/ports/redis"
	b_srv "github.com/Secreto31126/tesis/common/services/boards"
	p_srv "github.com/Secreto31126/tesis/common/services/postits"
	r_srv "github.com/Secreto31126/tesis/common/services/realtime"
	"github.com/gin-contrib/cors"
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

	executer := executer.New()

	boardService := b_srv.New(db)
	postitService := p_srv.New(db, cache, executer)
	realtimeService := r_srv.New(*boardService, cache)

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(cors.Default())

	boardController := boards.NewController(boardService, realtimeService)
	postitController := postits.NewController(postitService)

	api := router.Group("/v1")
	boardController.RegisterRoutes(api)
	postitController.RegisterRoutes(api)

	router.Run(ADDR)
}
