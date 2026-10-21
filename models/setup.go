// This is the process for the database connection
package models

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const connectionString = "mongodb://localhost:27017/goAPITest"
const db = "goAPI"
const collname = "users"

var mongoClient *mongo.Client

func ConnectDatabase() {
	clientOption := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(context.TODO(), clientOption)

	if err != nil {
		panic(err)
	}

	mongoClient = client

}
