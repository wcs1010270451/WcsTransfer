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
	GetUserBillingReconciliation(ctx context.Context) ([]entity.UserBillingReconciliation, error)
}

type Notifier interface {
	SendReconciliationMismatch(ctx context.Context, item entity.UserBillingReconciliation) error
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
	items, err := s.store.GetUserBillingReconciliation(ctx)
	if err != nil {
		log.Printf("reconciliation_check_failed: %v", err)
		return
	}

	for _, item := range items {
		if math.Abs(item.WalletVsLedgerDiff) < s.diffThreshold && math.Abs(item.LedgerVsLogsDiff) < s.diffThreshold {
			s.clearAlerted(item.UserID)
			continue
		}

		if s.markAlerted(item.UserID) {
			continue
		}

		log.Printf(
			"reconciliation_mismatch user_id=%d user_email=%q wallet_balance=%.4f ledger_net=%.4f wallet_vs_ledger_diff=%.4f ledger_debit=%.4f log_billable=%.4f ledger_vs_logs_diff=%.4f",
			item.UserID, item.UserEmail,
			item.WalletBalance, item.LedgerNetAmount, item.WalletVsLedgerDiff,
			item.LedgerDebitAmount, item.LogBillableAmount, item.LedgerVsLogsDiff,
		)
		if s.notifier != nil {
			if err := s.notifier.SendReconciliationMismatch(ctx, item); err != nil {
				log.Printf("reconciliation_alert_failed user_id=%d user_email=%q err=%v", item.UserID, item.UserEmail, err)
			}
		}
	}
}

func (s *Service) markAlerted(userID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.alerted[userID] {
		return true
	}
	s.alerted[userID] = true
	return false
}

func (s *Service) clearAlerted(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.alerted, userID)
}
