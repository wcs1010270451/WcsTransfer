package reconciliation

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"wcstransfer/backend/internal/entity"
)

type Store interface {
	GetTenantBillingReconciliation(ctx context.Context) ([]entity.TenantBillingReconciliation, error)
}

type Notifier interface {
	SendReconciliationMismatch(ctx context.Context, item entity.TenantBillingReconciliation) error
}

type Service struct {
	store         Store
	interval      time.Duration
	diffThreshold float64
	notifier      Notifier
	mu            sync.Mutex
	alerted       map[int64]bool
}

func New(store Store, interval time.Duration, diffThreshold float64, notifier Notifier) *Service {
	if interval <= 0 {
		interval = time.Hour
	}
	if diffThreshold <= 0 {
		diffThreshold = 0.0001
	}
	return &Service{
		store:         store,
		interval:      interval,
		diffThreshold: diffThreshold,
		notifier:      notifier,
		alerted:       make(map[int64]bool),
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
	items, err := s.store.GetTenantBillingReconciliation(ctx)
	if err != nil {
		log.Printf("reconciliation_check_failed: %v", err)
		return
	}

	for _, item := range items {
		if math.Abs(item.WalletVsLedgerDiff) < s.diffThreshold && math.Abs(item.LedgerVsLogsDiff) < s.diffThreshold {
			s.clearAlerted(item.TenantID)
			continue
		}

		if s.markAlerted(item.TenantID) {
			continue
		}

		log.Printf(
			"reconciliation_mismatch tenant_id=%d tenant_name=%q wallet_balance=%.4f ledger_net=%.4f wallet_vs_ledger_diff=%.4f ledger_debit=%.4f log_billable=%.4f ledger_vs_logs_diff=%.4f",
			item.TenantID,
			item.TenantName,
			item.WalletBalance,
			item.LedgerNetAmount,
			item.WalletVsLedgerDiff,
			item.LedgerDebitAmount,
			item.LogBillableAmount,
			item.LedgerVsLogsDiff,
		)
		if s.notifier != nil {
			if err := s.notifier.SendReconciliationMismatch(ctx, item); err != nil {
				log.Printf("reconciliation_alert_failed tenant_id=%d tenant_name=%q err=%v", item.TenantID, item.TenantName, err)
			}
		}
	}
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

func (s *Service) clearAlerted(tenantID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.alerted, tenantID)
}
