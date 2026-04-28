package billingalert

import (
	"context"
	"log"
	"sync"
	"time"

	"wcstransfer/backend/internal/entity"
)

type Store interface {
	GetUserBillingDebitAnomalies(ctx context.Context, since time.Time, minCount int, minBillableAmount float64) ([]entity.UserBillingDebitAnomaly, error)
}

type Notifier interface {
	SendUserBillingDebitAnomaly(ctx context.Context, item entity.UserBillingDebitAnomaly, window time.Duration) error
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
	items, err := s.store.GetUserBillingDebitAnomalies(ctx, time.Now().UTC().Add(-s.window), s.minCount, s.minBillableAmount)
	if err != nil {
		log.Printf("billing_anomaly_check_failed: %v", err)
		return
	}

	active := make(map[int64]struct{}, len(items))
	for _, item := range items {
		active[item.UserID] = struct{}{}
		if s.markAlerted(item.UserID) {
			continue
		}

		log.Printf(
			"user_billing_debit_anomaly user_id=%d user_email=%q missing_debit_count=%d missing_billable_amount=%.4f missing_cost_amount=%.4f window=%s",
			item.UserID, item.UserEmail,
			item.MissingDebitCount, item.MissingBillableAmount, item.MissingCostAmount, s.window,
		)
		if s.notifier != nil {
			if err := s.notifier.SendUserBillingDebitAnomaly(ctx, item, s.window); err != nil {
				log.Printf("user_billing_debit_alert_failed user_id=%d user_email=%q err=%v", item.UserID, item.UserEmail, err)
			}
		}
	}

	s.clearRecovered(active)
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

func (s *Service) clearRecovered(active map[int64]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for userID := range s.alerted {
		if _, ok := active[userID]; !ok {
			delete(s.alerted, userID)
		}
	}
}
