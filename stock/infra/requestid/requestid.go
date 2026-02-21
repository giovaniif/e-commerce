package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey struct{}

var key = ctxKey{}

func FromContext(ctx context.Context) string {
	if v := ctx.Value(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key, id)
}

func Generate() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
