package reconciliation

import (
	"context"
	"testing"
	"time"

	"wcstransfer/backend/internal/entity"
)

type stubStore struct {
	items []entity.TenantBillingReconciliation
}

type stubNotifier struct {
	calls int
	items []entity.TenantBillingReconciliation
}

func (s *stubStore) GetTenantBillingReconciliation(context.Context) ([]entity.TenantBillingReconciliation, error) {
	return s.items, nil
}

func (s *stubNotifier) SendReconciliationMismatch(_ context.Context, item entity.TenantBillingReconciliation) error {
	s.calls++
	s.items = append(s.items, item)
	return nil
}

func TestNewDefaults(t *testing.T) {
	service := New(&stubStore{}, 0, 0, nil)
	if service.interval != time.Hour {
		t.Fatalf("expected default interval 1h, got %s", service.interval)
	}
	if service.diffThreshold != 0.0001 {
		t.Fatalf("expected default diff threshold 0.0001, got %f", service.diffThreshold)
	}
}

func TestRunOnceSendsAlertForMismatch(t *testing.T) {
	notifier := &stubNotifier{}
	service := New(&stubStore{
		items: []entity.TenantBillingReconciliation{
			{
				TenantID:           1,
				TenantName:         "tenant-a",
				WalletVsLedgerDiff: 0.2,
				LedgerVsLogsDiff:   0,
			},
		},
	}, time.Hour, 0.0001, notifier)

	service.runOnce(context.Background())

	if notifier.calls != 1 {
		t.Fatalf("expected 1 alert call, got %d", notifier.calls)
	}
}

func TestRunOnceSkipsBalancedItems(t *testing.T) {
	notifier := &stubNotifier{}
	service := New(&stubStore{
		items: []entity.TenantBillingReconciliation{
			{
				TenantID:           1,
				TenantName:         "tenant-a",
				WalletVsLedgerDiff: 0.00001,
				LedgerVsLogsDiff:   0.00001,
			},
		},
	}, time.Hour, 0.0001, notifier)

	service.runOnce(context.Background())

	if notifier.calls != 0 {
		t.Fatalf("expected 0 alert calls, got %d", notifier.calls)
	}
}

func TestRunOnceAlertsOnlyOnceUntilRecovered(t *testing.T) {
	store := &stubStore{
		items: []entity.TenantBillingReconciliation{
			{
				TenantID:           1,
				TenantName:         "tenant-a",
				WalletVsLedgerDiff: 0.2,
			},
		},
	}
	notifier := &stubNotifier{}
	service := New(store, time.Hour, 0.0001, notifier)

	service.runOnce(context.Background())
	service.runOnce(context.Background())

	if notifier.calls != 1 {
		t.Fatalf("expected 1 alert before recovery, got %d", notifier.calls)
	}

	store.items = []entity.TenantBillingReconciliation{
		{
			TenantID:           1,
			TenantName:         "tenant-a",
			WalletVsLedgerDiff: 0,
			LedgerVsLogsDiff:   0,
		},
	}
	service.runOnce(context.Background())

	store.items = []entity.TenantBillingReconciliation{
		{
			TenantID:           1,
			TenantName:         "tenant-a",
			WalletVsLedgerDiff: 0.2,
		},
	}
	service.runOnce(context.Background())

	if notifier.calls != 2 {
		t.Fatalf("expected alert again after recovery, got %d", notifier.calls)
	}
}
