package infra

import (
	"fmt"
)

func NewTimeoutError(details string) error {
	return fmt.Errorf("timeout error: %s", details)
}

func NewNetworkError(details string) error {
	return fmt.Errorf("network error: %s", details)
}
