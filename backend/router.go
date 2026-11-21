package main

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// newRouter is the engine every route mounts on. Which proxies it trusts
// decides the client address the login and sign-up limits key on: Gin's
// default trusts anyone, so a caller could name any address in
// X-Forwarded-For and dodge the per-address limit.
func newRouter(cfg config) *gin.Engine {
	router := gin.Default()
	router.RedirectTrailingSlash = false
	if err := router.SetTrustedProxies(cfg.trustedProxies); err != nil {
		// loadConfig already refused anything that is not an IP or a CIDR.
		panic(fmt.Sprintf("trusted proxies %v: %v", cfg.trustedProxies, err))
	}
	router.Use(cors.New(corsConfig()))
	return router
}
