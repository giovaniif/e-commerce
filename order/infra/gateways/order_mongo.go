package gateways

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type orderRecord struct {
	IdempotencyKey string    `bson:"idempotency_key"`
	ItemId         int32     `bson:"item_id"`
	Quantity       int32     `bson:"quantity"`
	CreatedAt      time.Time `bson:"created_at"`
}

type OrderGatewayMongo struct {
	collection *mongo.Collection
}

func NewOrderGatewayMongo(client *mongo.Client) *OrderGatewayMongo {
	col := client.Database("order").Collection("orders")
	return &OrderGatewayMongo{collection: col}
}

func (g *OrderGatewayMongo) SaveOrder(ctx context.Context, idempotencyKey string, itemId int32, quantity int32) error {
	go func() {
		g.collection.InsertOne(context.Background(), orderRecord{
			IdempotencyKey: idempotencyKey,
			ItemId:         itemId,
			Quantity:       quantity,
			CreatedAt:      time.Now(),
		})
	}()
	return nil
}
