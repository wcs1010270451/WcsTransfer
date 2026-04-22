package dependencyalert

import (
	"context"
	"testing"
	"time"

	"wcstransfer/backend/internal/platform"
)

type stubChecker struct {
	results map[string]platform.CheckResult
}

type stubNotifier struct {
	calls []string
}

func (s *stubChecker) Health(context.Context) map[string]platform.CheckResult {
	return s.results
}

func (s *stubNotifier) SendDependencyAnomaly(_ context.Context, dependency string, _ string) error {
	s.calls = append(s.calls, dependency)
	return nil
}

func TestRunOnceAlertsOnlyOnceUntilRecovered(t *testing.T) {
	checker := &stubChecker{
		results: map[string]platform.CheckResult{
			"postgres": {Status: "down", Details: "ping timeout"},
		},
	}
	notifier := &stubNotifier{}
	service := New(checker, notifier, time.Minute)

	service.runOnce(context.Background())
	service.runOnce(context.Background())

	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 alert before recovery, got %d", len(notifier.calls))
	}

	checker.results = map[string]platform.CheckResult{
		"postgres": {Status: "up"},
	}
	service.runOnce(context.Background())

	checker.results = map[string]platform.CheckResult{
		"postgres": {Status: "down", Details: "ping timeout"},
	}
	service.runOnce(context.Background())

	if len(notifier.calls) != 2 {
		t.Fatalf("expected second alert after recovery, got %d", len(notifier.calls))
	}
}
