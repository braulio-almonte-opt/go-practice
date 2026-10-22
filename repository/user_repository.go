package repository

import (
	"context"
	"fmt"

	"github.com/braulio-almonte-opt/go-practice/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository interface {
	InsertUser(ctx context.Context, user models.User) error
	FindByUsername(ctx context.Context, username string) (models.User, error)
	FindByID(ctx context.Context, userID string) (models.User, error)
	FindAll(ctx context.Context) ([]models.User, error)
}

type MongoUserDAO struct {
	collection *mongo.Collection
}

func NewMongoUserDAO(client *mongo.Client, dbName, collectionName string) *MongoUserDAO {
	return &MongoUserDAO{
		collection: client.Database(dbName).Collection(collectionName),
	}
}

var _ UserRepository = (*MongoUserDAO)(nil)

func (dao *MongoUserDAO) InsertUser(ctx context.Context, user models.User) error {
	_, err := dao.collection.InsertOne(ctx, user)
	return err
}

func (dao *MongoUserDAO) FindByUsername(ctx context.Context, username string) (models.User, error) {
	var user models.User
	filter := bson.D{{Key: "username", Value: username}}
	err := dao.collection.FindOne(ctx, filter).Decode(&user)
	return user, err
}

func (dao *MongoUserDAO) FindByID(ctx context.Context, userID string) (models.User, error) {
	var user models.User
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return user, fmt.Errorf("%w: %v", models.ErrInvalidUserID, err)
	}

	filter := bson.M{"_id": id}
	err = dao.collection.FindOne(ctx, filter).Decode(&user)
	return user, err
}

func (dao *MongoUserDAO) FindAll(ctx context.Context) ([]models.User, error) {
	var users []models.User
	cursor, err := dao.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}
