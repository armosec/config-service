package main

import (
	"config-service/routes/login"
	"config-service/routes/prob"
	"config-service/routes/v1/admin"
	"config-service/routes/v1/attack_chains"
	"config-service/routes/v1/cluster"
	"config-service/routes/v1/collaboration_config"
	"config-service/routes/v1/customer"
	"config-service/routes/v1/customer_config"
	"config-service/routes/v1/framework"
	"config-service/routes/v1/posture_exception"
	"config-service/routes/v1/registry_cron_job"
	"config-service/routes/v1/repository"
	"config-service/routes/v1/runtime_incidents"
	"config-service/routes/v1/users_notifications_vulnerabilities"

	"config-service/routes/v1/users_notifications_cache"
	"config-service/routes/v1/vulnerability_exception"
	"config-service/utils"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

func main() {
	//initialize and deffer shutdown
	defer initialize()()
	//Create routes
	router := setupRouter()
	//Start server (blocking)
	startServer(router)
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.ContextWithFallback = true
	//readiness and liveness probes
	prob.AddRoutes(router)

	//general middlewares
	//open telemetry middleware
	router.Use(otelgin.Middleware("config-service"))
	//response trace headers middleware
	router.Use(traceAttributesNHeader)
	//set a logger per request context with common fields
	router.Use(requestLoggerWithFields)
	//log request summary after served
	router.Use(requestSummary())
	//recover from panics with 500 response
	router.Use(ginzap.RecoveryWithZap(zapLogger, true))

	//Public routes

	//login routes
	login.AddRoutes(router)
	//public (not authenticate routes
	customer.AddPublicRoutes(router)

	//auth middleware
	router.Use(authenticate)

	//add protected routes
	admin.AddRoutes(router)
	cluster.AddRoutes(router)
	posture_exception.AddRoutes(router)
	vulnerability_exception.AddRoutes(router)
	customer_config.AddRoutes(router)
	customer.AddRoutes(router)
	framework.AddRoutes(router)
	repository.AddRoutes(router)
	registry_cron_job.AddRoutes(router)
	collaboration_config.AddRoutes(router)
	users_notifications_cache.AddRoutes(router)
	users_notifications_vulnerabilities.AddRoutes(router)
	attack_chains.AddRoutes(router)
	runtime_incidents.AddRoutes(router)

	return router
}

func startServer(handler http.Handler) {
	port := utils.GetConfig().Port
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}
	zapLogger.Info("Starting server on port " + port)

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLogger.Info("Shutting down server...")
	// let the server have 5 secs to shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown:", zap.Error(err))
		return
	}
	zapLogger.Info("Server exiting")
}
