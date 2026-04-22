package billingalert

import (
	"context"
	"log"
	"sync"
	"time"

	"wcstransfer/backend/internal/entity"
)

type Store interface {
	GetTenantBillingDebitAnomalies(ctx context.Context, since time.Time, minCount int, minBillableAmount float64) ([]entity.TenantBillingDebitAnomaly, error)
}

type Notifier interface {
	SendTenantBillingDebitAnomaly(ctx context.Context, item entity.TenantBillingDebitAnomaly, window time.Duration) error
}

type Service struct {
	store             Store
	notifier          Notifier
	window            time.Duration
	interval          time.Duration
	minCount          int
	minBillableAmount float64
	mu                sync.Mutex
	alerted           map[int64]bool
}

func New(store Store, notifier Notifier, window time.Duration, interval time.Duration, minCount int, minBillableAmount float64) *Service {
	if window <= 0 {
		window = 10 * time.Minute
	}
	if interval <= 0 {
		interval = time.Minute
	}
	if minCount <= 0 {
		minCount = 1
	}
	if minBillableAmount <= 0 {
		minBillableAmount = 0.01
	}

	return &Service{
		store:             store,
		notifier:          notifier,
		window:            window,
		interval:          interval,
		minCount:          minCount,
		minBillableAmount: minBillableAmount,
		alerted:           make(map[int64]bool),
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
	items, err := s.store.GetTenantBillingDebitAnomalies(ctx, time.Now().UTC().Add(-s.window), s.minCount, s.minBillableAmount)
	if err != nil {
		log.Printf("billing_anomaly_check_failed: %v", err)
		return
	}

	active := make(map[int64]struct{}, len(items))
	for _, item := range items {
		active[item.TenantID] = struct{}{}
		if s.markAlerted(item.TenantID) {
			continue
		}

		log.Printf(
			"tenant_billing_debit_anomaly tenant_id=%d tenant_name=%q missing_debit_count=%d missing_billable_amount=%.4f missing_cost_amount=%.4f window=%s",
			item.TenantID,
			item.TenantName,
			item.MissingDebitCount,
			item.MissingBillableAmount,
			item.MissingCostAmount,
			s.window,
		)
		if s.notifier != nil {
			if err := s.notifier.SendTenantBillingDebitAnomaly(ctx, item, s.window); err != nil {
				log.Printf("tenant_billing_debit_alert_failed tenant_id=%d tenant_name=%q err=%v", item.TenantID, item.TenantName, err)
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
