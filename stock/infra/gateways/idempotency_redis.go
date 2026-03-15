package gateways

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

type reserveResult struct {
	ReservationId int32   `json:"reservation_id"`
	TotalFee      float64 `json:"total_fee"`
}

type IdempotencyGatewayRedis struct {
	client *redis.Client
}

func NewIdempotencyGatewayRedis(client *redis.Client) *IdempotencyGatewayRedis {
	return &IdempotencyGatewayRedis{client: client}
}

// ReserveIdempotency checks if a reserve request with this requestId was already processed.
// Returns (reservationId, totalFee, true, nil) if a cached result exists.
// Returns (0, 0, false, nil) if this is a new request.
func (g *IdempotencyGatewayRedis) ReserveIdempotency(ctx context.Context, requestId string) (int32, float64, bool, error) {
	key := fmt.Sprintf("stock:reserve:%s", requestId)
	data, err := g.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, fmt.Errorf("redis get: %w", err)
	}
	var result reserveResult
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, 0, false, fmt.Errorf("unmarshal: %w", err)
	}
	return result.ReservationId, result.TotalFee, true, nil
}

// SaveReserveResult caches the result of a successful reserve call.
func (g *IdempotencyGatewayRedis) SaveReserveResult(ctx context.Context, requestId string, reservationId int32, totalFee float64) error {
	key := fmt.Sprintf("stock:reserve:%s", requestId)
	raw, err := json.Marshal(reserveResult{ReservationId: reservationId, TotalFee: totalFee})
	if err != nil {
		return err
	}
	return g.client.Set(ctx, key, raw, idempotencyTTL).Err()
}

// ReleaseIdempotency returns true if this reservationId was already released.
func (g *IdempotencyGatewayRedis) ReleaseIdempotency(ctx context.Context, reservationId int32) (bool, error) {
	key := fmt.Sprintf("stock:release:%d", reservationId)
	exists, err := g.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return exists > 0, nil
}

// SaveReleaseResult marks a reservationId as released.
func (g *IdempotencyGatewayRedis) SaveReleaseResult(ctx context.Context, reservationId int32) error {
	key := fmt.Sprintf("stock:release:%d", reservationId)
	return g.client.Set(ctx, key, "1", idempotencyTTL).Err()
}

// CompleteIdempotency returns true if this reservationId was already completed.
func (g *IdempotencyGatewayRedis) CompleteIdempotency(ctx context.Context, reservationId int32) (bool, error) {
	key := fmt.Sprintf("stock:complete:%d", reservationId)
	exists, err := g.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return exists > 0, nil
}

// SaveCompleteResult marks a reservationId as completed.
func (g *IdempotencyGatewayRedis) SaveCompleteResult(ctx context.Context, reservationId int32) error {
	key := fmt.Sprintf("stock:complete:%d", reservationId)
	return g.client.Set(ctx, key, "1", idempotencyTTL).Err()
}
