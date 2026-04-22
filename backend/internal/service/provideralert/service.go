package provideralert

import (
	"context"
	"log"
	"sync"
	"time"

	"wcstransfer/backend/internal/entity"
)

type Store interface {
	GetProviderRequestAnomalies(ctx context.Context, since time.Time, minRequests int, rateLimitedThreshold float64, serverErrorThreshold float64) ([]entity.ProviderRequestAnomaly, error)
}

type Notifier interface {
	SendProviderRequestAnomaly(ctx context.Context, item entity.ProviderRequestAnomaly, window time.Duration) error
}

type Service struct {
	store                Store
	notifier             Notifier
	window               time.Duration
	interval             time.Duration
	minRequests          int
	rateLimitedThreshold float64
	serverErrorThreshold float64
	mu                   sync.Mutex
	alerted              map[int64]bool
}

func New(store Store, notifier Notifier, window time.Duration, interval time.Duration, minRequests int, rateLimitedThreshold float64, serverErrorThreshold float64) *Service {
	if window <= 0 {
		window = 5 * time.Minute
	}
	if interval <= 0 {
		interval = time.Minute
	}
	if minRequests <= 0 {
		minRequests = 10
	}
	if rateLimitedThreshold <= 0 {
		rateLimitedThreshold = 0.2
	}
	if serverErrorThreshold <= 0 {
		serverErrorThreshold = 0.2
	}

	return &Service{
		store:                store,
		notifier:             notifier,
		window:               window,
		interval:             interval,
		minRequests:          minRequests,
		rateLimitedThreshold: rateLimitedThreshold,
		serverErrorThreshold: serverErrorThreshold,
		alerted:              make(map[int64]bool),
	}
}

func (s *Service) Start(ctx context.Context) {
	if s == nil || s.store == nil {
		return
	}

	go func() {
		s.runOnce(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *Service) runOnce(ctx context.Context) {
	items, err := s.store.GetProviderRequestAnomalies(
		ctx,
		time.Now().UTC().Add(-s.window),
		s.minRequests,
		s.rateLimitedThreshold,
		s.serverErrorThreshold,
	)
	if err != nil {
		log.Printf("provider_anomaly_check_failed: %v", err)
		return
	}

	active := make(map[int64]struct{}, len(items))
	for _, item := range items {
		active[item.ProviderID] = struct{}{}
		if s.markAlerted(item.ProviderID) {
			continue
		}

		log.Printf(
			"provider_request_anomaly provider_id=%d provider_name=%q total_requests=%d rate_limited_count=%d server_error_count=%d rate_limited_ratio=%.4f server_error_ratio=%.4f window=%s",
			item.ProviderID,
			item.ProviderName,
			item.TotalRequests,
			item.RateLimitedCount,
			item.ServerErrorCount,
			item.RateLimitedRatio,
			item.ServerErrorRatio,
			s.window,
		)
		if s.notifier != nil {
			if err := s.notifier.SendProviderRequestAnomaly(ctx, item, s.window); err != nil {
				log.Printf("provider_anomaly_alert_failed provider_id=%d provider_name=%q err=%v", item.ProviderID, item.ProviderName, err)
			}
		}
	}

	s.clearRecovered(active)
}

func (s *Service) markAlerted(providerID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.alerted[providerID] {
		return true
	}
	s.alerted[providerID] = true
	return false
}

func (s *Service) clearRecovered(active map[int64]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for providerID := range s.alerted {
		if _, ok := active[providerID]; !ok {
			delete(s.alerted, providerID)
		}
	}
}
