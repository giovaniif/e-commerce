package gateways

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type PaymentGatewayHttp struct {}

func NewPaymentGatewayHttp() *PaymentGatewayHttp {
	return &PaymentGatewayHttp{}
}

type ChargeRequest struct {
	Amount float64 `json:"amount"`
}

func (p *PaymentGatewayHttp) Charge(amount float64) error {
  payload := ChargeRequest{
    Amount: amount,
  }
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("failed to marshal payload")
		return err
	}

  url := "http://localhost:3132/charge"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("failed to create request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
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