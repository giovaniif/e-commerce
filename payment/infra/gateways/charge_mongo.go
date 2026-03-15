package gateways

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type chargeRecord struct {
	Amount    float64   `bson:"amount"`
	CreatedAt time.Time `bson:"created_at"`
}

type ChargeGatewayMongo struct {
	collection *mongo.Collection
}

func NewChargeGatewayMongo(client *mongo.Client) *ChargeGatewayMongo {
	col := client.Database("payment").Collection("charges")
	return &ChargeGatewayMongo{collection: col}
}

func (g *ChargeGatewayMongo) Charge(amount float64) error {
	go func() {
		g.collection.InsertOne(context.Background(), chargeRecord{
			Amount:    amount,
			CreatedAt: time.Now(),
		})
	}()
	return nil
}
