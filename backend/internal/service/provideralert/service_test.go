package provideralert

import (
	"context"
	"testing"
	"time"

	"wcstransfer/backend/internal/entity"
)

type stubStore struct {
	items []entity.ProviderRequestAnomaly
}

type stubNotifier struct {
	calls int
	items []entity.ProviderRequestAnomaly
}

func (s *stubStore) GetProviderRequestAnomalies(context.Context, time.Time, int, float64, float64) ([]entity.ProviderRequestAnomaly, error) {
	return s.items, nil
}

func (s *stubNotifier) SendProviderRequestAnomaly(_ context.Context, item entity.ProviderRequestAnomaly, _ time.Duration) error {
	s.calls++
	s.items = append(s.items, item)
	return nil
}

func TestRunOnceAlertsOnlyOnceUntilRecovered(t *testing.T) {
	store := &stubStore{
		items: []entity.ProviderRequestAnomaly{
			{ProviderID: 1, ProviderName: "p1", TotalRequests: 20, RateLimitedCount: 8, RateLimitedRatio: 0.4, IsRateLimitedAnomalous: true},
		},
	}
	notifier := &stubNotifier{}
	service := New(store, notifier, 5*time.Minute, time.Minute, 10, 0.2, 0.2)

	service.runOnce(context.Background())
	service.runOnce(context.Background())

	if notifier.calls != 1 {
		t.Fatalf("expected 1 alert before recovery, got %d", notifier.calls)
	}

	store.items = nil
	service.runOnce(context.Background())

	store.items = []entity.ProviderRequestAnomaly{
		{ProviderID: 1, ProviderName: "p1", TotalRequests: 20, ServerErrorCount: 10, ServerErrorRatio: 0.5, IsServerErrorAnomalous: true},
	}
	service.runOnce(context.Background())

	if notifier.calls != 2 {
		t.Fatalf("expected second alert after recovery, got %d", notifier.calls)
	}
}
