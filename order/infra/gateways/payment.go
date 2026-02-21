package gateways

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/giovaniif/e-commerce/order/infra/requestid"
	"github.com/giovaniif/e-commerce/order/infra/tracing"
)

type PaymentGatewayHttp struct {
	httpClient *http.Client
	baseURL    string
}

func NewPaymentGatewayHttp(httpClient *http.Client, baseURL string) *PaymentGatewayHttp {
	return &PaymentGatewayHttp{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

type ChargeRequest struct {
	Amount float64 `json:"amount"`
}

func (p *PaymentGatewayHttp) Charge(ctx context.Context, amount float64, idempotencyKey string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	payload := ChargeRequest{
		Amount: amount,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("failed to marshal payload")
		return err
	}

	reqURL, _ := url.JoinPath(p.baseURL, "charge")
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("failed to create request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	if id := requestid.FromContext(ctx); id != "" {
		req.Header.Set("X-Request-ID", id)
	}
	tracing.Inject(ctx, req.Header)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		fmt.Println("failed to do request")
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to charge")
	}
	return nil
}
