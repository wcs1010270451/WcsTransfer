package clientquota

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"wcstransfer/backend/internal/entity"
)

var ErrLimitExceeded = errors.New("client quota exceeded")

type Violation struct {
	Type    string
	Message string
	Limit   int
	Current int64
	ResetAt time.Time
}

func (v *Violation) Error() string {
	if v == nil {
		return ""
	}
	return v.Message
}

type Service struct {
	redis *redis.Client
}

func New(redisClient *redis.Client) *Service {
	return &Service{redis: redisClient}
}

func (s *Service) ConsumeRequest(ctx context.Context, clientKey entity.ClientAPIKey) error {
	if s == nil || s.redis == nil || clientKey.ID <= 0 {
		return nil
	}

	now := time.Now()
	if clientKey.RPMLimit > 0 {
		key := fmt.Sprintf("quota:client:%d:rpm:%s", clientKey.ID, now.UTC().Format("200601021504"))
		count, err := s.incrementWithTTL(ctx, key, time.Minute)
		if err != nil {
			return err
		}
		if count > int64(clientKey.RPMLimit) {
			return &Violation{
				Type:    "rpm_limit_exceeded",
				Message: "client rpm limit exceeded",
				Limit:   clientKey.RPMLimit,
				Current: count,
				ResetAt: now.UTC().Truncate(time.Minute).Add(time.Minute),
			}
		}
	}

	if clientKey.DailyRequestLimit > 0 {
		key := fmt.Sprintf("quota:client:%d:daily_requests:%s", clientKey.ID, now.UTC().Format("20060102"))
		count, err := s.incrementWithTTL(ctx, key, ttlUntilTomorrowUTC(now))
		if err != nil {
			return err
		}
		if count > int64(clientKey.DailyRequestLimit) {
			return &Violation{
				Type:    "daily_request_limit_exceeded",
				Message: "client daily request limit exceeded",
				Limit:   clientKey.DailyRequestLimit,
				Current: count,
				ResetAt: tomorrowUTC(now),
			}
		}
	}

	if clientKey.DailyTokenLimit > 0 {
		key := fmt.Sprintf("quota:client:%d:daily_tokens:%s", clientKey.ID, now.UTC().Format("20060102"))
		current, err := s.redis.Get(ctx, key).Int64()
		if err != nil && !errors.Is(err, redis.Nil) {
			return err
		}
		if current >= int64(clientKey.DailyTokenLimit) {
			return &Violation{
				Type:    "daily_token_limit_exceeded",
				Message: "client daily token limit exceeded",
				Limit:   clientKey.DailyTokenLimit,
				Current: current,
				ResetAt: tomorrowUTC(now),
			}
		}
	}

	return nil
}

func (s *Service) AddTokenUsage(ctx context.Context, clientKey entity.ClientAPIKey, totalTokens int) error {
	if s == nil || s.redis == nil || clientKey.ID <= 0 || totalTokens <= 0 {
		return nil
	}

	now := time.Now()
	key := fmt.Sprintf("quota:client:%d:daily_tokens:%s", clientKey.ID, now.UTC().Format("20060102"))
	_, err := s.incrementByWithTTL(ctx, key, int64(totalTokens), ttlUntilTomorrowUTC(now))
	return err
}

func (s *Service) incrementWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return s.incrementByWithTTL(ctx, key, 1, ttl)
}

func (s *Service) incrementByWithTTL(ctx context.Context, key string, amount int64, ttl time.Duration) (int64, error) {
	pipe := s.redis.TxPipeline()
	incr := pipe.IncrBy(ctx, key, amount)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func tomorrowUTC(now time.Time) time.Time {
	utc := now.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day()+1, 0, 0, 0, 0, time.UTC)
}

func ttlUntilTomorrowUTC(now time.Time) time.Duration {
	return time.Until(tomorrowUTC(now))
}
