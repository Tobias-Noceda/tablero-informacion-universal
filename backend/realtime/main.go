package main

import (
	"context"

	"github.com/Secreto31126/tesis/common/controllers/realtime"
	"github.com/Secreto31126/tesis/common/middleware"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	srv "github.com/Secreto31126/tesis/common/services/realtime"

	"github.com/gin-gonic/gin"
)

const (
	ADDR = "0.0.0.0:62113"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rt, err := mongo.New()
	if err != nil {
		panic(err)
	}
	defer rt.Close()

	r_srv := srv.New(rt, ctx)

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	controller := realtime.NewController(r_srv)

	api := router.Group("/v1")
	controller.RegisterRoutes(api)

	router.Run(ADDR)
}
