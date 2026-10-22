// This is the process for the database connection
package models

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const connectionString = "mongodb://localhost:27017/goAPITest"
const DatabaseName = "goAPI"
const UserCollectionName = "users"

func ConnectDatabase() (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return connectDatabase(ctx, connectionString)
}

func connectDatabase(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("create MongoDB client: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if disconnectErr := client.Disconnect(cleanupCtx); disconnectErr != nil {
			return nil, fmt.Errorf("ping MongoDB: %w (disconnect after failed ping: %v)", err, disconnectErr)
		}
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	return client, nil
}
