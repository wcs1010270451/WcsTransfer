package billingalert

import (
	"context"
	"testing"
	"time"

	"wcstransfer/backend/internal/entity"
)

type stubStore struct {
	items []entity.UserBillingDebitAnomaly
}

type stubNotifier struct {
	calls int
	items []entity.UserBillingDebitAnomaly
}

func (s *stubStore) GetUserBillingDebitAnomalies(context.Context, time.Time, int, float64) ([]entity.UserBillingDebitAnomaly, error) {
	return s.items, nil
}

func (s *stubNotifier) SendUserBillingDebitAnomaly(_ context.Context, item entity.UserBillingDebitAnomaly, _ time.Duration) error {
	s.calls++
	s.items = append(s.items, item)
	return nil
}

func TestRunOnceAlertsOnlyOnceUntilRecovered(t *testing.T) {
	store := &stubStore{
		items: []entity.UserBillingDebitAnomaly{
			{UserID: 1, UserEmail: "user-a@test.com", MissingDebitCount: 2, MissingBillableAmount: 1.2, IsCountAnomalous: true},
		},
	}
	notifier := &stubNotifier{}
	service := New(store, notifier, 10*time.Minute, time.Minute, 1, 0.01)

	service.runOnce(context.Background())
	service.runOnce(context.Background())

	if notifier.calls != 1 {
		t.Fatalf("expected 1 alert before recovery, got %d", notifier.calls)
	}

	store.items = nil
	service.runOnce(context.Background())

	store.items = []entity.UserBillingDebitAnomaly{
		{UserID: 1, UserEmail: "user-a@test.com", MissingDebitCount: 1, MissingBillableAmount: 0.5, IsCountAnomalous: true},
	}
	service.runOnce(context.Background())

	if notifier.calls != 2 {
		t.Fatalf("expected second alert after recovery, got %d", notifier.calls)
	}
}
