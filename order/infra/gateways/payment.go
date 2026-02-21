package gateways

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type PaymentGatewayHttp struct {
	httpClient *http.Client
}

func NewPaymentGatewayHttp(httpClient *http.Client) *PaymentGatewayHttp {
	return &PaymentGatewayHttp{
		httpClient: httpClient,
	}
}

type ChargeRequest struct {
	Amount float64 `json:"amount"`
}

func (p *PaymentGatewayHttp) Charge(ctx context.Context, amount float64) error {
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

	// url := "http://payment:3132/charge"
	url := "http://localhost:3132/charge"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("failed to create request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
