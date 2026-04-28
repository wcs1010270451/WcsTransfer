package alerting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"wcstransfer/backend/internal/entity"
)

func sampleItem() entity.UserBillingReconciliation {
	return entity.UserBillingReconciliation{
		UserID:           7,
		UserEmail:         "user-a@test.com",
		WalletBalance:      10.5,
		LedgerCreditAmount: 20,
		LedgerDebitAmount:  9.5,
		LedgerNetAmount:    10.5,
		LogBillableAmount:  9.6,
		LogCostAmount:      4.2,
		WalletVsLedgerDiff: 0,
		LedgerVsLogsDiff:   -0.1,
	}
}

func TestWebhookNotifierWecomPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected application/json, got %s", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, "wecom", time.Second)
	if err := notifier.SendReconciliationMismatch(context.Background(), sampleItem()); err != nil {
		t.Fatalf("send webhook: %v", err)
	}

	if payload["msgtype"] != "text" {
		t.Fatalf("expected wecom text payload, got %+v", payload)
	}
}

func TestWebhookNotifierGenericPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, "generic", time.Second)
	notifier.now = func() time.Time {
		return time.Date(2026, 4, 22, 2, 0, 0, 0, time.UTC)
	}
	if err := notifier.SendReconciliationMismatch(context.Background(), sampleItem()); err != nil {
		t.Fatalf("send webhook: %v", err)
	}

	if payload["event"] != "reconciliation_mismatch" {
		t.Fatalf("expected generic reconciliation event, got %+v", payload)
	}
}

func TestWebhookNotifierFeishuProviderAnomalyPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, "feishu", time.Second)
	err := notifier.SendProviderRequestAnomaly(context.Background(), entity.ProviderRequestAnomaly{
		ProviderID:             2,
		ProviderName:           "qwen",
		TotalRequests:          20,
		RateLimitedCount:       6,
		RateLimitedRatio:       0.3,
		IsRateLimitedAnomalous: true,
	}, 5*time.Minute)
	if err != nil {
		t.Fatalf("send webhook: %v", err)
	}

	if payload["msg_type"] != "text" {
		t.Fatalf("expected feishu text payload, got %+v", payload)
	}
}

func TestWebhookNotifierWecomWalletBlockPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, "wecom", time.Second)
	err := notifier.SendUserWalletBlockAnomaly(context.Background(), entity.UserWalletBlockAnomaly{
		UserID:                  1,
		UserEmail:                "king",
		WalletBlockedCount:        7,
		ReserveBlockedCount:       2,
		IsWalletBlockedAnomalous:  true,
		IsReserveBlockedAnomalous: false,
	}, 5*time.Minute)
	if err != nil {
		t.Fatalf("send webhook: %v", err)
	}

	if payload["msgtype"] != "text" {
		t.Fatalf("expected wecom text payload, got %+v", payload)
	}
}

func TestWebhookNotifierFeishuBillingDebitPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, "feishu", time.Second)
	err := notifier.SendUserBillingDebitAnomaly(context.Background(), entity.UserBillingDebitAnomaly{
		UserID:                  1,
		UserEmail:                "king",
		MissingDebitCount:         2,
		MissingBillableAmount:     1.23,
		MissingCostAmount:         0.67,
		IsCountAnomalous:          true,
		IsBillableAmountAnomalous: true,
	}, 10*time.Minute)
	if err != nil {
		t.Fatalf("send webhook: %v", err)
	}

	if payload["msg_type"] != "text" {
		t.Fatalf("expected feishu text payload, got %+v", payload)
	}
}

func TestWebhookNotifierGenericDependencyPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, "generic", time.Second)
	err := notifier.SendDependencyAnomaly(context.Background(), "postgres", "ping timeout")
	if err != nil {
		t.Fatalf("send webhook: %v", err)
	}

	if payload["event"] != "dependency_anomaly" {
		t.Fatalf("expected dependency anomaly event, got %+v", payload)
	}
}
