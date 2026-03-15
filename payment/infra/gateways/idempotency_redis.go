package gateways

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	protocols "github.com/giovaniif/e-commerce/payment/protocols"
)

const (
	chargeIdempotencyKeyPrefix = "idempotency:charge:"
	chargeIdempotencyTTL       = 24 * time.Hour
)

type idempotencyRedisState struct {
	Status string                          `json:"status"`
	Result *protocols.IdempotencyKeyResult `json:"result,omitempty"`
}

type IdempotencyGatewayRedis struct {
	client *redis.Client
}

func NewIdempotencyGatewayRedis(client *redis.Client) *IdempotencyGatewayRedis {
	return &IdempotencyGatewayRedis{client: client}
}

func (g *IdempotencyGatewayRedis) key(k string) string {
	return chargeIdempotencyKeyPrefix + k
}

func (g *IdempotencyGatewayRedis) ReserveIdempotencyKey(idempotencyKey string) (*protocols.IdempotencyKeyResult, error) {
	ctx := context.Background()
	k := g.key(idempotencyKey)

	for {
		data, err := g.client.Get(ctx, k).Bytes()
		if err == redis.Nil {
			state := idempotencyRedisState{Status: "processing"}
			raw, _ := json.Marshal(state)
			_, err := g.client.SetArgs(ctx, k, raw, redis.SetArgs{Mode: "NX", TTL: chargeIdempotencyTTL}).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("redis set: %w", err)
			}
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("redis get: %w", err)
		}

		var state idempotencyRedisState
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		switch state.Status {
		case "success":
			return state.Result, nil
		case "processing":
			return nil, errors.New("idempotency key is already being processed")
		default:
			_ = g.client.Del(ctx, k).Err()
			newState := idempotencyRedisState{Status: "processing"}
			raw, _ := json.Marshal(newState)
			if err := g.client.Set(ctx, k, raw, chargeIdempotencyTTL).Err(); err != nil {
				return nil, fmt.Errorf("redis set: %w", err)
			}
			return nil, nil
		}
	}
}

func (g *IdempotencyGatewayRedis) MarkFailure(idempotencyKey string) error {
	return g.client.Del(context.Background(), g.key(idempotencyKey)).Err()
}

func (g *IdempotencyGatewayRedis) MarkSuccess(idempotencyKey string) error {
	state := idempotencyRedisState{
		Status: "success",
		Result: &protocols.IdempotencyKeyResult{Success: true},
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return g.client.Set(context.Background(), g.key(idempotencyKey), raw, chargeIdempotencyTTL).Err()
}
