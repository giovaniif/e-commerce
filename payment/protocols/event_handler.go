package protocols

import "context"

type EventHandler interface {
	Handle(ctx context.Context, eventType string, payload []byte, traceparent string) error
}
