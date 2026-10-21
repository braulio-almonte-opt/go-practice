package models

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserName string             `json:"username"`
	Password string             `json:"password"`
	RealName string             `json:"realname"`
	LastName string             `json:"lastname"`
	Balance  int                `json:"balance"`
	Role     string             `json:"role"`
}

func InsertUser(user User) error {
	collection := mongoClient.Database(db).Collection(collname)
	inserted, err := collection.InsertOne(context.TODO(), user)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Inserted a record with id: ", inserted.InsertedID)
	return err
}

func InsertManyUsers(users []User) error {
	newUsers := make([]interface{}, len(users))
	for i, user := range users {
		newUsers[i] = user
	}
	collection := mongoClient.Database(db).Collection(collname)
	result, err := collection.InsertMany(context.TODO(), newUsers)

	if err != nil {
		log.Fatal(err)
	}
	log.Println(result)

	return err
}

func UpdateUser(userID string, user User) error {
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return (err)
	}

	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{
		"username": user.UserName,
		"password": user.Password,
		"realname": user.RealName,
		"lastname": user.LastName,
		"role":     user.Role,
		"balance":  user.Balance}}

	collection := mongoClient.Database(db).Collection(collname)
	result, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}
	fmt.Println("New record", result)
	return nil
}

func DeleteUser(userID string) error {
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": id}
	collection := mongoClient.Database(db).Collection(collname)
	result, err := collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}

	fmt.Println("Deleted object: ", result)
	return nil
}

func Find(userName string) User {
	var result User

	filter := bson.D{{"username", userName}}
	collection := mongoClient.Database(db).Collection(collname)
	err := collection.FindOne(context.TODO(), filter).Decode(&result)

	if err != nil {
		log.Fatal(err)
	}
	return result
}

func FindAll() []User {

	var results []User
	filter := bson.M{}
	collection := mongoClient.Database(db).Collection(collname)
	cursor, err := collection.Find(context.TODO(), filter)

	if err != nil {
		log.Fatal(err)
	}

	err = cursor.All(context.TODO(), &results)
	if err != nil {
		log.Fatal(err)
	}

	return results
}
