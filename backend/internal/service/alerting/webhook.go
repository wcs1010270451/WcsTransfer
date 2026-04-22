package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"wcstransfer/backend/internal/entity"
)

type WebhookNotifier struct {
	url      string
	provider string
	client   *http.Client
	now      func() time.Time
}

func NewWebhookNotifier(url string, provider string, timeout time.Duration) *WebhookNotifier {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	if normalizedProvider == "" {
		normalizedProvider = "generic"
	}

	return &WebhookNotifier{
		url:      strings.TrimSpace(url),
		provider: normalizedProvider,
		client:   &http.Client{Timeout: timeout},
		now:      time.Now,
	}
}

func (n *WebhookNotifier) SendReconciliationMismatch(ctx context.Context, item entity.TenantBillingReconciliation) error {
	if n == nil || n.url == "" {
		return nil
	}

	body, err := n.buildPayload(item)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *WebhookNotifier) SendProviderRequestAnomaly(ctx context.Context, item entity.ProviderRequestAnomaly, window time.Duration) error {
	if n == nil || n.url == "" {
		return nil
	}

	body, err := n.buildProviderAnomalyPayload(item, window)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *WebhookNotifier) SendTenantWalletBlockAnomaly(ctx context.Context, item entity.TenantWalletBlockAnomaly, window time.Duration) error {
	if n == nil || n.url == "" {
		return nil
	}

	body, err := n.buildTenantWalletBlockPayload(item, window)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *WebhookNotifier) SendTenantBillingDebitAnomaly(ctx context.Context, item entity.TenantBillingDebitAnomaly, window time.Duration) error {
	if n == nil || n.url == "" {
		return nil
	}

	body, err := n.buildTenantBillingDebitPayload(item, window)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *WebhookNotifier) SendDependencyAnomaly(ctx context.Context, dependency string, details string) error {
	if n == nil || n.url == "" {
		return nil
	}

	body, err := n.buildDependencyAnomalyPayload(dependency, details)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *WebhookNotifier) buildPayload(item entity.TenantBillingReconciliation) ([]byte, error) {
	content := fmt.Sprintf(
		"[WcsTransfer] 对账异常 tenant_id=%d tenant_name=%s wallet_balance=%.4f ledger_net=%.4f wallet_vs_ledger_diff=%.4f ledger_debit=%.4f log_billable=%.4f ledger_vs_logs_diff=%.4f",
		item.TenantID,
		item.TenantName,
		item.WalletBalance,
		item.LedgerNetAmount,
		item.WalletVsLedgerDiff,
		item.LedgerDebitAmount,
		item.LogBillableAmount,
		item.LedgerVsLogsDiff,
	)

	switch n.provider {
	case "wecom":
		return json.Marshal(map[string]any{
			"msgtype": "text",
			"text": map[string]string{
				"content": content,
			},
		})
	case "feishu":
		return json.Marshal(map[string]any{
			"msg_type": "text",
			"content": map[string]string{
				"text": content,
			},
		})
	default:
		return json.Marshal(map[string]any{
			"event":     "reconciliation_mismatch",
			"level":     "error",
			"source":    "wcstransfer.reconciliation",
			"message":   content,
			"timestamp": n.now().UTC().Format(time.RFC3339),
			"data": map[string]any{
				"tenant_id":             item.TenantID,
				"tenant_name":           item.TenantName,
				"wallet_balance":        item.WalletBalance,
				"ledger_credit_amount":  item.LedgerCreditAmount,
				"ledger_debit_amount":   item.LedgerDebitAmount,
				"ledger_net_amount":     item.LedgerNetAmount,
				"log_billable_amount":   item.LogBillableAmount,
				"log_cost_amount":       item.LogCostAmount,
				"wallet_vs_ledger_diff": item.WalletVsLedgerDiff,
				"ledger_vs_logs_diff":   item.LedgerVsLogsDiff,
			},
		})
	}
}

func (n *WebhookNotifier) buildProviderAnomalyPayload(item entity.ProviderRequestAnomaly, window time.Duration) ([]byte, error) {
	content := fmt.Sprintf(
		"[WcsTransfer] Provider 异常 provider_id=%d provider_name=%s window=%s total_requests=%d rate_limited_count=%d server_error_count=%d rate_limited_ratio=%.2f%% server_error_ratio=%.2f%%",
		item.ProviderID,
		item.ProviderName,
		window,
		item.TotalRequests,
		item.RateLimitedCount,
		item.ServerErrorCount,
		item.RateLimitedRatio*100,
		item.ServerErrorRatio*100,
	)

	switch n.provider {
	case "wecom":
		return json.Marshal(map[string]any{
			"msgtype": "text",
			"text": map[string]string{
				"content": content,
			},
		})
	case "feishu":
		return json.Marshal(map[string]any{
			"msg_type": "text",
			"content": map[string]string{
				"text": content,
			},
		})
	default:
		return json.Marshal(map[string]any{
			"event":     "provider_request_anomaly",
			"level":     "error",
			"source":    "wcstransfer.provider",
			"message":   content,
			"timestamp": n.now().UTC().Format(time.RFC3339),
			"data": map[string]any{
				"provider_id":               item.ProviderID,
				"provider_name":             item.ProviderName,
				"window":                    window.String(),
				"total_requests":            item.TotalRequests,
				"rate_limited_count":        item.RateLimitedCount,
				"server_error_count":        item.ServerErrorCount,
				"rate_limited_ratio":        item.RateLimitedRatio,
				"server_error_ratio":        item.ServerErrorRatio,
				"is_rate_limited_anomalous": item.IsRateLimitedAnomalous,
				"is_server_error_anomalous": item.IsServerErrorAnomalous,
			},
		})
	}
}

func (n *WebhookNotifier) buildTenantWalletBlockPayload(item entity.TenantWalletBlockAnomaly, window time.Duration) ([]byte, error) {
	content := fmt.Sprintf(
		"[WcsTransfer] 钱包拦截异常 tenant_id=%d tenant_name=%s window=%s wallet_blocked_count=%d reserve_blocked_count=%d",
		item.TenantID,
		item.TenantName,
		window,
		item.WalletBlockedCount,
		item.ReserveBlockedCount,
	)

	switch n.provider {
	case "wecom":
		return json.Marshal(map[string]any{
			"msgtype": "text",
			"text": map[string]string{
				"content": content,
			},
		})
	case "feishu":
		return json.Marshal(map[string]any{
			"msg_type": "text",
			"content": map[string]string{
				"text": content,
			},
		})
	default:
		return json.Marshal(map[string]any{
			"event":     "tenant_wallet_block_anomaly",
			"level":     "error",
			"source":    "wcstransfer.wallet",
			"message":   content,
			"timestamp": n.now().UTC().Format(time.RFC3339),
			"data": map[string]any{
				"tenant_id":                    item.TenantID,
				"tenant_name":                  item.TenantName,
				"window":                       window.String(),
				"wallet_blocked_count":         item.WalletBlockedCount,
				"reserve_blocked_count":        item.ReserveBlockedCount,
				"is_wallet_blocked_anomalous":  item.IsWalletBlockedAnomalous,
				"is_reserve_blocked_anomalous": item.IsReserveBlockedAnomalous,
			},
		})
	}
}

func (n *WebhookNotifier) buildTenantBillingDebitPayload(item entity.TenantBillingDebitAnomaly, window time.Duration) ([]byte, error) {
	content := fmt.Sprintf(
		"[WcsTransfer] 扣费异常 tenant_id=%d tenant_name=%s window=%s missing_debit_count=%d missing_billable_amount=%.4f missing_cost_amount=%.4f",
		item.TenantID,
		item.TenantName,
		window,
		item.MissingDebitCount,
		item.MissingBillableAmount,
		item.MissingCostAmount,
	)

	switch n.provider {
	case "wecom":
		return json.Marshal(map[string]any{
			"msgtype": "text",
			"text": map[string]string{
				"content": content,
			},
		})
	case "feishu":
		return json.Marshal(map[string]any{
			"msg_type": "text",
			"content": map[string]string{
				"text": content,
			},
		})
	default:
		return json.Marshal(map[string]any{
			"event":     "tenant_billing_debit_anomaly",
			"level":     "error",
			"source":    "wcstransfer.billing",
			"message":   content,
			"timestamp": n.now().UTC().Format(time.RFC3339),
			"data": map[string]any{
				"tenant_id":                    item.TenantID,
				"tenant_name":                  item.TenantName,
				"window":                       window.String(),
				"missing_debit_count":          item.MissingDebitCount,
				"missing_billable_amount":      item.MissingBillableAmount,
				"missing_cost_amount":          item.MissingCostAmount,
				"is_count_anomalous":           item.IsCountAnomalous,
				"is_billable_amount_anomalous": item.IsBillableAmountAnomalous,
			},
		})
	}
}

func (n *WebhookNotifier) buildDependencyAnomalyPayload(dependency string, details string) ([]byte, error) {
	content := fmt.Sprintf(
		"[WcsTransfer] 依赖异常 dependency=%s details=%s",
		dependency,
		strings.TrimSpace(details),
	)

	switch n.provider {
	case "wecom":
		return json.Marshal(map[string]any{
			"msgtype": "text",
			"text": map[string]string{
				"content": content,
			},
		})
	case "feishu":
		return json.Marshal(map[string]any{
			"msg_type": "text",
			"content": map[string]string{
				"text": content,
			},
		})
	default:
		return json.Marshal(map[string]any{
			"event":     "dependency_anomaly",
			"level":     "error",
			"source":    "wcstransfer.health",
			"message":   content,
			"timestamp": n.now().UTC().Format(time.RFC3339),
			"data": map[string]any{
				"dependency": dependency,
				"details":    strings.TrimSpace(details),
			},
		})
	}
}
