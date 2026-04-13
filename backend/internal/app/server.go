package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"wcstransfer/backend/internal/config"
	"wcstransfer/backend/internal/platform"
	"wcstransfer/backend/internal/router"
)

type Server struct {
	config config.Config
}

func NewServer(cfg config.Config) *Server {
	return &Server{config: cfg}
}

func (s *Server) Run(ctx context.Context) error {
	deps, err := platform.New(ctx, s.config)
	if err != nil {
		return fmt.Errorf("initialize dependencies: %w", err)
	}
	defer deps.Close()

	engine := router.New(s.config, deps, nil)

	srv := &http.Server{
		Addr:         s.config.Address(),
		Handler:      engine,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		log.Printf("starting %s on %s", s.config.AppName, s.config.Address())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}

		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()

		log.Printf("shutting down %s", s.config.AppName)
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		return <-errCh
	}
}
