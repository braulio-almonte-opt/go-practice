package models

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidUserID = errors.New("invalid user ID")

type User struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserName string             `json:"username"`
	Password string             `json:"password"`
	RealName string             `json:"realname"`
	LastName string             `json:"lastname"`
	Balance  float64            `json:"balance"`
	Role     string             `json:"role"`
}

//This is for the future
// type UserRepository interface {
// 	InsertUser(ctx context.Context, user User) error
// }

// type MongoUserDAO struct {
// 	client *mongo.Client
// 	dbName string
// 	coll   string
// }

// func NewMongoUserDAO(client *mongo.Client, dbName, coll string) *MongoUserDAO {
// 	return &MongoUserDAO{client: client, dbName: dbName, coll: coll}
// }

// func (dao *MongoUserDAO) InsertUser2(ctx context.Context, user User) error {
// 	collection := dao.client.Database(dao.dbName).Collection(dao.coll)

// 	_, err := collection.InsertOne(ctx, user)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func InsertUser(user User) error {
	collection := mongoClient.Database(db).Collection(collname)
	inserted, err := collection.InsertOne(context.TODO(), user)

	if err != nil {
		log.Default().Println(err)
		return err
	}

	fmt.Println("Inserted a record with id: ", inserted.InsertedID)
	return err
}

func Find(userName string) (User, error) {
	var result User

	filter := bson.D{{Key: "username", Value: userName}}
	collection := mongoClient.Database(db).Collection(collname)
	err := collection.FindOne(context.TODO(), filter).Decode(&result)

	if err != nil {
		log.Default().Println(err)
		return result, err
	}

	return result, err
}

func FindByID(userID string) (User, error) {

	var result User
	id, err := primitive.ObjectIDFromHex(userID)

	if err != nil {
		return result, fmt.Errorf("%w: %v", ErrInvalidUserID, err)
	}

	filter := bson.M{"_id": id}

	collection := mongoClient.Database(db).Collection(collname)

	err = collection.FindOne(context.TODO(), filter).Decode(&result)

	if err != nil {
		return result, err
	}

	return result, err

}

func FindAll() ([]User, error) {

	var results []User
	filter := bson.M{}
	collection := mongoClient.Database(db).Collection(collname)
	cursor, err := collection.Find(context.TODO(), filter)

	if err != nil {
		log.Default().Println(err)
		return nil, err
	}

	err = cursor.All(context.TODO(), &results)
	if err != nil {
		log.Default().Println(err)
		return nil, err
	}

	return results, err
}
