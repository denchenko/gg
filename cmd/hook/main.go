package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/denchenko/gg/internal/adapters"
	httpadapter "github.com/denchenko/gg/internal/adapters/primary/http"
	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core"
	do "github.com/samber/do/v2"
)

func main() {
	injector := do.New(
		config.Package,
		core.Package,
		adapters.SecondaryPackage,
		adapters.PrimaryPackage,
	)

	server, err := do.Invoke[*httpadapter.Server](injector)
	if err != nil {
		log.Fatalf("Failed to create HTTP server: %v", err)
	}

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
