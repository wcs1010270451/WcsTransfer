package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"wcstransfer/backend/internal/config"
	"wcstransfer/backend/internal/platform"
	repopostgres "wcstransfer/backend/internal/repository/postgres"
	"wcstransfer/backend/internal/router"
	"wcstransfer/backend/internal/service/alerting"
	"wcstransfer/backend/internal/service/billingalert"
	"wcstransfer/backend/internal/service/dependencyalert"
	"wcstransfer/backend/internal/service/provideralert"
	"wcstransfer/backend/internal/service/reconciliation"
	"wcstransfer/backend/internal/service/walletalert"
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

	var notifier *alerting.WebhookNotifier
	if s.config.AlertWebhookURL != "" {
		notifier = alerting.NewWebhookNotifier(
			s.config.AlertWebhookURL,
			s.config.AlertWebhookProvider,
			s.config.AlertWebhookTimeout,
		)
	}

	if deps.Postgres != nil {
		store := repopostgres.NewStore(deps.Postgres)
		if err := store.EnsureBootstrapAdmin(ctx, s.config.AdminBootstrapUsername, s.config.AdminBootstrapPassword, s.config.AdminBootstrapDisplayName); err != nil {
			return fmt.Errorf("ensure bootstrap admin: %w", err)
		}
		if s.config.ReconciliationEnabled {
			reconciliation.New(store, s.config.ReconciliationInterval, s.config.ReconciliationDiffThreshold, notifier).Start(ctx)
		}
		if s.config.ProviderAlertEnabled {
			provideralert.New(
				store,
				notifier,
				s.config.ProviderAlertWindow,
				s.config.ProviderAlertInterval,
				s.config.ProviderAlertMinRequests,
				s.config.ProviderAlert429Threshold,
				s.config.ProviderAlert5xxThreshold,
			).Start(ctx)
		}
		if s.config.TenantWalletAlertEnabled {
			walletalert.New(
				store,
				notifier,
				s.config.TenantWalletAlertWindow,
				s.config.TenantWalletAlertInterval,
				s.config.TenantWalletAlertMinBlocks,
				s.config.TenantReserveAlertMinBlocks,
			).Start(ctx)
		}
		if s.config.BillingAlertEnabled {
			billingalert.New(
				store,
				notifier,
				s.config.BillingAlertWindow,
				s.config.BillingAlertInterval,
				s.config.BillingAlertMinCount,
				s.config.BillingAlertMinAmount,
			).Start(ctx)
		}
	}
	if s.config.DependencyAlertEnabled {
		dependencyalert.New(
			deps,
			notifier,
			s.config.DependencyAlertInterval,
		).Start(ctx)
	}

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
