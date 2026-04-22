package dependencyalert

import (
	"context"
	"log"
	"sync"
	"time"

	"wcstransfer/backend/internal/platform"
)

type HealthChecker interface {
	Health(ctx context.Context) map[string]platform.CheckResult
}

type Notifier interface {
	SendDependencyAnomaly(ctx context.Context, dependency string, details string) error
}

type Service struct {
	checker  HealthChecker
	notifier Notifier
	interval time.Duration
	mu       sync.Mutex
	alerted  map[string]bool
}

func New(checker HealthChecker, notifier Notifier, interval time.Duration) *Service {
	if interval <= 0 {
		interval = time.Minute
	}

	return &Service{
		checker:  checker,
		notifier: notifier,
		interval: interval,
		alerted:  make(map[string]bool),
	}
}

func (s *Service) Start(ctx context.Context) {
	if s == nil || s.checker == nil {
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
	results := s.checker.Health(ctx)
	active := make(map[string]struct{}, len(results))

	for dependency, result := range results {
		if result.Status != "down" {
			s.clearAlerted(dependency)
			continue
		}

		active[dependency] = struct{}{}
		if s.markAlerted(dependency) {
			continue
		}

		log.Printf("dependency_anomaly dependency=%q details=%q", dependency, result.Details)
		if s.notifier != nil {
			if err := s.notifier.SendDependencyAnomaly(ctx, dependency, result.Details); err != nil {
				log.Printf("dependency_alert_failed dependency=%q err=%v", dependency, err)
			}
		}
	}

	s.clearRecovered(active)
}

func (s *Service) markAlerted(dependency string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.alerted[dependency] {
		return true
	}
	s.alerted[dependency] = true
	return false
}

func (s *Service) clearAlerted(dependency string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.alerted, dependency)
}

func (s *Service) clearRecovered(active map[string]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for dependency := range s.alerted {
		if _, ok := active[dependency]; !ok {
			delete(s.alerted, dependency)
		}
	}
}
