package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/reingma/Unframed/backend/internal/api"
	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/publish"
	"github.com/reingma/Unframed/backend/internal/rebuild"
	"github.com/reingma/Unframed/backend/internal/router"
	"github.com/reingma/Unframed/backend/internal/snapshot"
	"github.com/reingma/Unframed/backend/internal/subscriber/logs"
	"github.com/reingma/Unframed/backend/internal/watch"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	logger := slog.Default()
	addr := os.Getenv("BACKEND_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	root := os.Getenv("CONTENT_ROOT")
	if root == "" {
		return errors.New("CONTENT_ROOT is required: set it to the content directory")
	}
	source, err := content.OpenFS(root)
	if err != nil {
		return fmt.Errorf("open CONTENT_ROOT: %w", err)
	}
	defer source.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	store := &snapshot.Store{}
	publisher := publish.NewLocal(logs.Subscriber{Logger: logger})
	manager := rebuild.New(store, rebuild.FromSource(source, publisher, logger), logger)
	if err := manager.Start(ctx); err != nil {
		return err
	}
	managerDone := make(chan struct{})
	go func() { manager.Run(ctx); close(managerDone) }()
	defer func() { cancel(); <-managerDone }()

	result := make(chan error, 2)
	go func() {
		if err := manager.Follow(ctx, watch.New(root, watch.Debounce, logger)); err != nil {
			result <- err
		}
	}()

	handlers := api.New(store, manager,
		api.NewToken(os.Getenv("PREVIEW_TOKEN")), api.NewToken(os.Getenv("ADMIN_TOKEN")), logger)
	server := &http.Server{
		Addr:              addr,
		Handler:           api.Observe(router.New(handlers), logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info("starting backend", "address", addr, "content_root", root)
		result <- server.ListenAndServe()
	}()
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		_ = server.Close()
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}
