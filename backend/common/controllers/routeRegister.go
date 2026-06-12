package controllers

import "github.com/gin-gonic/gin"

type RouteRegistrar interface {
	RegisterRoutes(router gin.IRouter)
}
