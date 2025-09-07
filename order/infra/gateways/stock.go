package gateways

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
	ItemId int32 `json:"itemId"`
	Quantity int32 `json:"quantity"`
}

type ReleaseRequest struct {
	ReservationId string `json:"reservationId"`
}

type CompleteRequest struct {
	ReservationId string `json:"reservationId"`
}

func (s *StockGatewayHttp) Reserve(itemId int32, quantity int32) (*protocols.Reservation, error) {
  url := "http://localhost:3133/reserve"
  payload := ReserveRequest{
    ItemId: itemId,
    Quantity: quantity,
  }
  payloadBytes, err := json.Marshal(payload)
  if err != nil {
    fmt.Println("failed to marshal payload")
    return nil, err
  }
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("failed to create request")
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Println("failed to do request")
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to reserve stock")
	}
	var reservation protocols.Reservation
	err = json.NewDecoder(resp.Body).Decode(&reservation)
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

func (s *StockGatewayHttp) Release(reservationId string) error {
	url := "http://localhost:3133/release"
  payload := ReleaseRequest{
    ReservationId: reservationId,
  }
  payloadBytes, err := json.Marshal(payload)
  if err != nil {
    fmt.Println("failed to marshal payload")
    return err
  }
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
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

func (s *StockGatewayHttp) Complete(reservationId string) error {
	url := "http://localhost:3133/complete"
  payload := CompleteRequest{
    ReservationId: reservationId,
  }
  payloadBytes, err := json.Marshal(payload)
  if err != nil {
    fmt.Println("failed to marshal payload")
    return err
  }
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
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
		return errors.New("failed to complete stock")
	}
	return nil
}