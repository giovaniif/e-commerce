package gateways

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

const (
	idempotencyKeyPrefix = "idempotency:checkout:"
	idempotencyTTL       = 24 * time.Hour
)

type checkoutRedisState struct {
	Status string                              `json:"status"`
	Result *protocols.CheckoutIdempotencyKeyResult `json:"result,omitempty"`
}

type CheckoutGatewayRedis struct {
	client *redis.Client
}

func NewCheckoutGatewayRedis(client *redis.Client) *CheckoutGatewayRedis {
	return &CheckoutGatewayRedis{client: client}
}

func (c *CheckoutGatewayRedis) key(idempotencyKey string) string {
	return idempotencyKeyPrefix + idempotencyKey
}

func (c *CheckoutGatewayRedis) ReserveIdempotencyKey(ctx context.Context, idempotencyKey string) (*protocols.CheckoutIdempotencyKeyResult, error) {
	k := c.key(idempotencyKey)

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		data, err := c.client.Get(ctx, k).Bytes()
		if err == redis.Nil {
			state := checkoutRedisState{Status: "processing"}
			raw, _ := json.Marshal(state)
			_, err := c.client.SetArgs(ctx, k, raw, redis.SetArgs{Mode: "NX", TTL: idempotencyTTL}).Result()
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

		var state checkoutRedisState
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, fmt.Errorf("redis unmarshal: %w", err)
		}

		switch state.Status {
		case "success":
			return state.Result, nil
		case "processing":
			return nil, errors.New("idempotency key is already being processed")
		default:
			_ = c.client.Del(ctx, k).Err()
			newState := checkoutRedisState{Status: "processing"}
			raw, _ := json.Marshal(newState)
			if err := c.client.Set(ctx, k, raw, idempotencyTTL).Err(); err != nil {
				return nil, fmt.Errorf("redis set: %w", err)
			}
			return nil, nil
		}
	}
}

func (c *CheckoutGatewayRedis) MarkFailure(ctx context.Context, idempotencyKey string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return c.client.Del(ctx, c.key(idempotencyKey)).Err()
}

func (c *CheckoutGatewayRedis) MarkSuccess(ctx context.Context, idempotencyKey string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	state := checkoutRedisState{
		Status: "success",
		Result: &protocols.CheckoutIdempotencyKeyResult{Success: true, Error: nil},
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(idempotencyKey), raw, idempotencyTTL).Err()
}
