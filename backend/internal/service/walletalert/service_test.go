package walletalert

import (
	"context"
	"testing"
	"time"

	"wcstransfer/backend/internal/entity"
)

type stubStore struct {
	items []entity.UserWalletBlockAnomaly
}

type stubNotifier struct {
	calls int
	items []entity.UserWalletBlockAnomaly
}

func (s *stubStore) GetUserWalletBlockAnomalies(context.Context, time.Time, int, int) ([]entity.UserWalletBlockAnomaly, error) {
	return s.items, nil
}

func (s *stubNotifier) SendUserWalletBlockAnomaly(_ context.Context, item entity.UserWalletBlockAnomaly, _ time.Duration) error {
	s.calls++
	s.items = append(s.items, item)
	return nil
}

func TestRunOnceAlertsOnlyOnceUntilRecovered(t *testing.T) {
	store := &stubStore{
		items: []entity.UserWalletBlockAnomaly{
			{UserID: 1, UserEmail: "user-a@test.com", WalletBlockedCount: 10, IsWalletBlockedAnomalous: true},
		},
	}
	notifier := &stubNotifier{}
	service := New(store, notifier, 5*time.Minute, time.Minute, 5, 5)

	service.runOnce(context.Background())
	service.runOnce(context.Background())

	if notifier.calls != 1 {
		t.Fatalf("expected 1 alert before recovery, got %d", notifier.calls)
	}

	store.items = nil
	service.runOnce(context.Background())

	store.items = []entity.UserWalletBlockAnomaly{
		{UserID: 1, UserEmail: "user-a@test.com", ReserveBlockedCount: 8, IsReserveBlockedAnomalous: true},
	}
	service.runOnce(context.Background())

	if notifier.calls != 2 {
		t.Fatalf("expected second alert after recovery, got %d", notifier.calls)
	}
}
