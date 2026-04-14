package keyhealth

import (
	"sync"
	"time"

	"wcstransfer/backend/internal/entity"
)

type State struct {
	Until  time.Time
	Reason string
}

type Tracker struct {
	entries sync.Map
}

func NewTracker() *Tracker {
	return &Tracker{}
}

func (t *Tracker) Current(keyID int64, now time.Time) (State, bool) {
	if t == nil || keyID <= 0 {
		return State{}, false
	}

	if raw, ok := t.entries.Load(keyID); ok {
		state, valid := raw.(State)
		if valid && state.Until.After(now) {
			return state, true
		}
		if valid {
			t.entries.Delete(keyID)
		}
	}

	return State{}, false
}

func (t *Tracker) Penalize(keyID int64, duration time.Duration, reason string) {
	if t == nil || keyID <= 0 || duration <= 0 {
		return
	}

	t.entries.Store(keyID, State{
		Until:  time.Now().Add(duration),
		Reason: reason,
	})
}

func (t *Tracker) Clear(keyID int64) {
	if t == nil || keyID <= 0 {
		return
	}

	t.entries.Delete(keyID)
}

func (t *Tracker) EnrichKeys(items []entity.ProviderKey) []entity.ProviderKey {
	if t == nil {
		return items
	}

	now := time.Now()
	enriched := make([]entity.ProviderKey, len(items))
	copy(enriched, items)

	for index := range enriched {
		enriched[index].HealthStatus = "healthy"

		state, ok := t.Current(enriched[index].ID, now)
		if !ok {
			continue
		}

		until := state.Until.UTC()
		enriched[index].HealthStatus = "cooldown"
		enriched[index].CooldownReason = state.Reason
		enriched[index].CooldownUntil = &until
	}

	return enriched
}
