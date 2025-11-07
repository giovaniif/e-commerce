package gateways

import (
	"errors"
	"sync"

	protocols "github.com/giovaniif/e-commerce/payment/protocols"
)

type IdempotencyGatewayMemory struct {
	mutex           sync.RWMutex
	idempotencyKeys map[string]*IdempotencyState
}

type IdempotencyState struct {
	Status string
	Result *protocols.IdempotencyKeyResult
}

func NewIdempotencyGatewayMemory() *IdempotencyGatewayMemory {
	return &IdempotencyGatewayMemory{
		idempotencyKeys: make(map[string]*IdempotencyState),
	}
}

func (c *IdempotencyGatewayMemory) ReserveIdempotencyKey(idempotencyKey string) (*protocols.IdempotencyKeyResult, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	state, exists := c.idempotencyKeys[idempotencyKey]
	if exists {
		if state.Status == "success" {
			return state.Result, nil
		}

		if state.Status == "processing" {
			return nil, errors.New("idempotency key is already being processed")
		}

		delete(c.idempotencyKeys, idempotencyKey)
	}

	c.idempotencyKeys[idempotencyKey] = &IdempotencyState{
		Status: "processing",
	}
	return nil, nil
}

func (c *IdempotencyGatewayMemory) MarkFailure(idempotencyKey string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.idempotencyKeys, idempotencyKey)
	return nil
}

func (c *IdempotencyGatewayMemory) MarkSuccess(idempotencyKey string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if state, exists := c.idempotencyKeys[idempotencyKey]; exists {
		state.Status = "success"
		state.Result = &protocols.IdempotencyKeyResult{
			Success: true,
			Error:   nil,
		}
	}

	return nil
}
