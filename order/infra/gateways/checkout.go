package gateways

import (
	"errors"
	"sync"

	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

type CheckoutGatewayMemory struct {
	mutex           sync.RWMutex
	idempotencyKeys map[string]*ChekoutState
}

type ChekoutState struct {
	Status string
	Result *protocols.CheckoutIdempotencyKeyResult
}

func NewCheckoutGatewayMemory() *CheckoutGatewayMemory {
	return &CheckoutGatewayMemory{
		idempotencyKeys: make(map[string]*ChekoutState),
	}
}

func (c *CheckoutGatewayMemory) ReserveIdempotencyKey(idempotencyKey string) (*protocols.CheckoutIdempotencyKeyResult, error) {
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

	c.idempotencyKeys[idempotencyKey] = &ChekoutState{
		Status: "processing",
	}
	return nil, nil
}

func (c *CheckoutGatewayMemory) MarkFailure(idempotencyKey string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.idempotencyKeys, idempotencyKey)
	return nil
}

func (c *CheckoutGatewayMemory) MarkSuccess(idempotencyKey string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if state, exists := c.idempotencyKeys[idempotencyKey]; exists {
		state.Status = "success"
		state.Result = &protocols.CheckoutIdempotencyKeyResult{
			Success: true,
			Error:   nil,
		}
	}

	return nil
}
