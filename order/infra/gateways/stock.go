package gateways

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	infra "github.com/giovaniif/e-commerce/order/infra"
	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

type StockGatewayHttp struct {
	httpClient *http.Client
}

func NewStockGatewayHttp(httpClient *http.Client) *StockGatewayHttp {
	return &StockGatewayHttp{
		httpClient: httpClient,
	}
}

type ReserveRequest struct {
	ItemId   int32 `json:"itemId"`
	Quantity int32 `json:"quantity"`
}

type ReleaseRequest struct {
	ReservationId int32 `json:"reservationId"`
}

type CompleteRequest struct {
	ReservationId int32 `json:"reservationId"`
}

type ReservationResponse struct {
	ReservationId int32   `json:"reservationId"`
	TotalFee      float64 `json:"totalFee"`
}

func (s *StockGatewayHttp) Reserve(ctx context.Context, itemId int32, quantity int32) (*protocols.Reservation, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// url := "http://stock:3133/reserve"
	url := "http://localhost:3133/reserve"
	payload := ReserveRequest{
		ItemId:   itemId,
		Quantity: quantity,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusGatewayTimeout {
		return nil, infra.NewTimeoutError("timeout reserving stock")
	}
	if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		return nil, infra.NewNetworkError("network error reserving stock")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to reserve stock")
	}
	var reservation ReservationResponse
	err = json.NewDecoder(resp.Body).Decode(&reservation)
	if err != nil {
		return nil, err
	}
	return &protocols.Reservation{
		Id:       reservation.ReservationId,
		TotalFee: reservation.TotalFee,
	}, nil
}

func (s *StockGatewayHttp) Release(ctx context.Context, reservationId int32) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// url := "http://stock:3133/release"
	url := "http://localhost:3133/release"
	payload := ReleaseRequest{
		ReservationId: reservationId,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("failed to marshal payload")
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("failed to create request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Println("failed to do request")
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to release stock")
	}
	return nil
}

func (s *StockGatewayHttp) Complete(ctx context.Context, reservationId int32) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// url := "http://stock:3133/complete"
	url := "http://localhost:3133/complete"
	payload := CompleteRequest{
		ReservationId: reservationId,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("failed to marshal payload")
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("failed to create request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Println("failed to do request")
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("failed to complete stock")
		return errors.New("failed to complete stock")
	}
	return nil
}
