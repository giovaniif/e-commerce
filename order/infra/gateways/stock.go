package gateways

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	infra 	"github.com/giovaniif/e-commerce/order/infra"
	"github.com/giovaniif/e-commerce/order/infra/requestid"
	"github.com/giovaniif/e-commerce/order/infra/tracing"
	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

type StockGatewayHttp struct {
	httpClient *http.Client
	baseURL    string
}

func NewStockGatewayHttp(httpClient *http.Client, baseURL string) *StockGatewayHttp {
	return &StockGatewayHttp{
		httpClient: httpClient,
		baseURL:    baseURL,
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

	reqURL, _ := url.JoinPath(s.baseURL, "reserve")
	payload := ReserveRequest{
		ItemId:   itemId,
		Quantity: quantity,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if id := requestid.FromContext(ctx); id != "" {
		req.Header.Set("X-Request-ID", id)
	}
	tracing.Inject(ctx, req.Header)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reserve stock request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusGatewayTimeout {
		return nil, infra.NewTimeoutError("timeout reserving stock")
	}
	if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		return nil, infra.NewNetworkError("network error reserving stock")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to reserve stock (status %d): %s", resp.StatusCode, string(body))
	}
	var reservation ReservationResponse
	err = json.Unmarshal(body, &reservation)
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

	reqURL, _ := url.JoinPath(s.baseURL, "release")
	payload := ReleaseRequest{
		ReservationId: reservationId,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("failed to marshal payload")
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("failed to create request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if id := requestid.FromContext(ctx); id != "" {
		req.Header.Set("X-Request-ID", id)
	}
	tracing.Inject(ctx, req.Header)
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

	reqURL, _ := url.JoinPath(s.baseURL, "complete")
	payload := CompleteRequest{
		ReservationId: reservationId,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("failed to marshal payload")
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("failed to create request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if id := requestid.FromContext(ctx); id != "" {
		req.Header.Set("X-Request-ID", id)
	}
	tracing.Inject(ctx, req.Header)
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
