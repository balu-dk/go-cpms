package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/balu-dk/go-cpms/config"
	"github.com/balu-dk/go-cpms/internal/api"
	"github.com/balu-dk/go-cpms/internal/db"
	"github.com/balu-dk/go-cpms/internal/service"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	// Setup logger
	cfg.SetupLogger()
	logrus.Info("Starting CPMS server")

	// Connect to database
	store, err := db.NewPostgresStore(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to database")
	}
	defer store.Close()

	// Create CPMS service
	cpms := service.NewCPMS(cfg, store)

	// Start OCPP central system
	if err := cpms.Start(); err != nil {
		logrus.WithError(err).Fatal("Failed to start OCPP central system")
	}

	// Create API server
	apiServer := api.NewAPI(cpms)

	// Start API server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.APIPort),
		Handler: apiServer,
	}

	// Run the server in a goroutine
	go func() {
		logrus.Infof("Starting API server on port %d", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to start API server")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Create a deadline for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt to gracefully shut down the server
	if err := srv.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Server forced to shutdown")
	}

	logrus.Info("Server exited")
}
