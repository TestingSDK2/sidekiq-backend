package repository

import (
	"context"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type GroupMetaRepo struct {
	Db         *mongo.Database
	Collection *mongo.Collection
}

func NewChatMetaRepo(db *mongo.Database, collectionName string) domain.ChatMetaRepository {
	collection := db.Collection(collectionName)
	repo := &GroupMetaRepo{
		Db:         db,
		Collection: collection,
	}
	// _ = repo.RegisterGroupIndexes(context.TODO())
	return repo
}

func (gr GroupMetaRepo) Create(ctx context.Context, message domain.GroupMeta) (domain.GroupMeta, error) {
	doc, err := gr.Collection.InsertOne(ctx, message)
	if err != nil {
		logrus.Debug("Error while inserting group meta,Reason:", err)
		return domain.GroupMeta{}, err
	}

	insertedID, ok := doc.InsertedID.(primitive.ObjectID)
	if !ok {
		return domain.GroupMeta{}, ErrInvalidInsertedIDType
	}

	// Create a filter based on the _id field
	filter := bson.M{"_id": insertedID}

	// Perform the find operation
	var insertedM domain.GroupMeta
	err = gr.Collection.FindOne(ctx, filter).Decode(&insertedM)
	if err != nil {
		// Handle the error (document not found, decoding error, etc.)
		return insertedM, err
	}

	return insertedM, nil
}

func (gr GroupMetaRepo) GetByGroupMemberId(ctx context.Context, memberId int, groupId primitive.ObjectID) (domain.GroupMeta, error) {
	filter := bson.M{"groupId": groupId, "memberId": memberId}
	var group domain.GroupMeta
	err := gr.Collection.FindOne(ctx, filter).Decode(&group)
	if err != nil {
		return group, err
	}
	return group, nil
}

func (gr GroupMetaRepo) UpdateStartChatMessage(ctx context.Context, memberId int, groupId primitive.ObjectID, chatStartMessage primitive.ObjectID) error {
	filter := bson.M{"groupId": groupId, "memberId": memberId}
	update := bson.M{
		"$set": bson.M{
			"chatStartMessage": chatStartMessage,
			"updatedAt":        time.Now(),
		},
	}
	_, err := gr.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (gr GroupMetaRepo) UpdateLastSeenMessage(ctx context.Context, memberId int, groupId primitive.ObjectID, lastSeenMessageId primitive.ObjectID) error {
	logrus.Info(memberId, groupId)
	filter := bson.M{"groupId": groupId, "memberId": memberId}
	update := bson.M{
		"$set": bson.M{
			"lastSeenMessage": lastSeenMessageId,
			"updatedAt":       time.Now(),
		},
	}
	_, err := gr.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}
