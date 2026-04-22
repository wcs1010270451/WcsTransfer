package walletalert

import (
	"context"
	"log"
	"sync"
	"time"

	"wcstransfer/backend/internal/entity"
)

type Store interface {
	GetTenantWalletBlockAnomalies(ctx context.Context, since time.Time, walletBlockThreshold int, reserveBlockThreshold int) ([]entity.TenantWalletBlockAnomaly, error)
}

type Notifier interface {
	SendTenantWalletBlockAnomaly(ctx context.Context, item entity.TenantWalletBlockAnomaly, window time.Duration) error
}

type Service struct {
	store                 Store
	notifier              Notifier
	window                time.Duration
	interval              time.Duration
	walletBlockThreshold  int
	reserveBlockThreshold int
	mu                    sync.Mutex
	alerted               map[int64]bool
}

func New(store Store, notifier Notifier, window time.Duration, interval time.Duration, walletBlockThreshold int, reserveBlockThreshold int) *Service {
	if window <= 0 {
		window = 5 * time.Minute
	}
	if interval <= 0 {
		interval = time.Minute
	}
	if walletBlockThreshold <= 0 {
		walletBlockThreshold = 5
	}
	if reserveBlockThreshold <= 0 {
		reserveBlockThreshold = 5
	}

	return &Service{
		store:                 store,
		notifier:              notifier,
		window:                window,
		interval:              interval,
		walletBlockThreshold:  walletBlockThreshold,
		reserveBlockThreshold: reserveBlockThreshold,
		alerted:               make(map[int64]bool),
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
	items, err := s.store.GetTenantWalletBlockAnomalies(
		ctx,
		time.Now().UTC().Add(-s.window),
		s.walletBlockThreshold,
		s.reserveBlockThreshold,
	)
	if err != nil {
		log.Printf("tenant_wallet_alert_check_failed: %v", err)
		return
	}

	active := make(map[int64]struct{}, len(items))
	for _, item := range items {
		active[item.TenantID] = struct{}{}
		if s.markAlerted(item.TenantID) {
			continue
		}

		log.Printf(
			"tenant_wallet_block_anomaly tenant_id=%d tenant_name=%q wallet_blocked_count=%d reserve_blocked_count=%d window=%s",
			item.TenantID,
			item.TenantName,
			item.WalletBlockedCount,
			item.ReserveBlockedCount,
			s.window,
		)
		if s.notifier != nil {
			if err := s.notifier.SendTenantWalletBlockAnomaly(ctx, item, s.window); err != nil {
				log.Printf("tenant_wallet_block_alert_failed tenant_id=%d tenant_name=%q err=%v", item.TenantID, item.TenantName, err)
			}
		}
	}

	s.clearRecovered(active)
}

func (s *Service) markAlerted(tenantID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.alerted[tenantID] {
		return true
	}
	s.alerted[tenantID] = true
	return false
}

func (s *Service) clearRecovered(active map[int64]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tenantID := range s.alerted {
		if _, ok := active[tenantID]; !ok {
			delete(s.alerted, tenantID)
		}
	}
}
