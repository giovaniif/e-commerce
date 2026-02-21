package infra

import (
	"errors"
	"fmt"
)

var (
	ErrTimeout  = errors.New("timeout error")
	ErrNetwork  = errors.New("network error")
)

func NewTimeoutError(details string) error {
	return fmt.Errorf("%w: %s", ErrTimeout, details)
}

func NewNetworkError(details string) error {
	return fmt.Errorf("%w: %s", ErrNetwork, details)
}

// IsRetriable returns true if the error is timeout or network (5xx), so retry makes sense.
func IsRetriable(err error) bool {
	return err != nil && (errors.Is(err, ErrTimeout) || errors.Is(err, ErrNetwork))
}
