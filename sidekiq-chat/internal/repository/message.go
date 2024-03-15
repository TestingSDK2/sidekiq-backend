package repository

import (
	"context"
	"errors"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageRepo struct {
	Db         *mongo.Database
	Collection *mongo.Collection
}

func NewMessageRepo(db *mongo.Database, collectionName string) domain.MessageRepo {
	collection := db.Collection(collectionName)
	repo := &MessageRepo{
		Db:         db,
		Collection: collection,
	}
	err := repo.RegisterMessageIndexes(context.TODO())
	if err != nil {
		logrus.Error("Unable to register indexes")
		logrus.Error(err)
		return nil
	}
	return repo
}

// RegisterGroupIndexes creates a unique index on the slug field
func (mr MessageRepo) RegisterMessageIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			// Index on groupI field
			Keys:    bson.D{{Key: "groupId", Value: 1}},
			Options: options.Index().SetName("group_id_index"),
		},
		{
			// Index on senderId field
			Keys:    bson.D{{Key: "senderId", Value: 1}},
			Options: options.Index().SetName("sender_index"),
		},
		{
			// Index on timestamp field
			Keys:    bson.D{{Key: "createdAt", Value: 1}},
			Options: options.Index().SetName("timestamp_index"),
		},
	}

	// Create indexes
	_, err := mr.Collection.Indexes().CreateMany(context.Background(), indexes)
	if err != nil {
		return err
	}

	return nil
}

func (gr MessageRepo) Create(ctx context.Context, message domain.Message) (domain.Message, error) {
	doc, err := gr.Collection.InsertOne(ctx, message)
	logrus.Info(doc)
	if err != nil {
		logrus.Debug("Error while inserting message,Reason:", err)
		return domain.Message{}, err
	}

	insertedID, ok := doc.InsertedID.(primitive.ObjectID)
	if !ok {
		return domain.Message{}, ErrInvalidInsertedIDType
	}

	// Create a filter based on the _id field
	filter := bson.M{"_id": insertedID}

	// Perform the find operation
	var insertedM domain.Message
	err = gr.Collection.FindOne(ctx, filter).Decode(&insertedM)
	if err != nil {
		// Handle the error (document not found, decoding error, etc.)
		return domain.Message{}, err
	}

	return insertedM, nil
}

func (gr MessageRepo) Delete(ctx context.Context, groupId, messageId primitive.ObjectID) error {
	filter := bson.M{"_id": messageId, "groupId": groupId}
	update := bson.M{"$set": bson.M{"isDeleted": true}}

	res, err := gr.Collection.UpdateOne(context.TODO(), filter, update)
	if res.ModifiedCount < 1 {
		return errors.New("nothing to update")
	}

	return err
}

func (mr MessageRepo) GetGroupMessages(ctx context.Context, groupId primitive.ObjectID, lastViewedMessageID primitive.ObjectID, page, pageSize int) ([]domain.Message, error) {
	var messages []domain.Message
	// Define the query filter to include messages newer than the last viewed message, if it exists
	filter := bson.M{
		"groupId":   groupId,
		"isDeleted": bson.M{"$ne": true}, // Filter out deleted messages
	}
	if lastViewedMessageID != primitive.NilObjectID {
		filter["_id"] = bson.M{"$gt": lastViewedMessageID}
	}

	// Calculate the number of documents to skip based on pagination parameters
	skip := (page - 1) * pageSize

	// Define sort options to fetch messages in reverse chronological order
	sortOptions := options.Find().SetSort(bson.D{{Key: "_id", Value: -1}})

	// Define options to limit the number of documents fetched and to skip documents based on pagination parameters
	findOptions := options.Find().
		SetLimit(int64(pageSize)).
		SetSkip(int64(skip))

	// Query for messages belonging to the specified group, filtered by last viewed message ID (if available), sorted in reverse chronological order with pagination
	cursor, err := mr.Collection.Find(context.Background(), filter, findOptions, sortOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	// Iterate through the cursor and decode each message
	for cursor.Next(context.Background()) {
		var message domain.Message
		if err := cursor.Decode(&message); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	// Check for any errors during cursor iteration
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (mr MessageRepo) GetMessageById(ctx context.Context, groupId, messageId primitive.ObjectID) (domain.Message, error) {
	filter := bson.M{
		"groupId": groupId,
		"_id":     messageId,
	}

	var message domain.Message
	err := mr.Collection.FindOne(ctx, filter).Decode(&message)
	if err != nil {
		return message, err
	}
	return message, nil
}
