package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrInvalidInsertedIDType = errors.New("invalid InsertedID type")

type GroupRepo struct {
	Db         *mongo.Database
	Collection *mongo.Collection
}

func NewGroupRepo(db *mongo.Database, collectionName string) domain.GroupRepository {
	collection := db.Collection(collectionName)
	repo := &GroupRepo{
		Db:         db,
		Collection: collection,
	}
	_ = repo.RegisterGroupIndexes(context.TODO())
	return repo
}

// RegisterGroupIndexes creates a unique index on the slug field
func (gr *GroupRepo) RegisterGroupIndexes(ctx context.Context) error {
	indexOptions := options.Index().SetUnique(true)
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: indexOptions,
	}

	_, err := gr.Collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (gr *GroupRepo) Create(ctx context.Context, group domain.Group) (domain.Group, error) {
	doc, err := gr.Collection.InsertOne(ctx, group)
	if err != nil {
		logrus.Debug("Error while inserting group,Readon:", err)
		return domain.Group{}, err
	}

	insertedID, ok := doc.InsertedID.(primitive.ObjectID)
	if !ok {
		return domain.Group{}, ErrInvalidInsertedIDType
	}

	// Create a filter based on the _id field
	filter := bson.M{"_id": insertedID}

	// Perform the find operation
	var insertedG domain.Group
	err = gr.Collection.FindOne(ctx, filter).Decode(&insertedG)
	if err != nil {
		// Handle the error (document not found, decoding error, etc.)
		return domain.Group{}, err
	}

	return insertedG, nil
}

func (gr GroupRepo) AddMember(ctx context.Context, groupID primitive.ObjectID, member domain.GroupMember) error {
	filter := bson.M{"_id": groupID, "isDeleted": false}
	update := bson.M{"$push": bson.M{"members": member}}

	_, err := gr.Collection.UpdateOne(context.TODO(), filter, update)
	return err
}

func (gr GroupRepo) RemoveMember(ctx context.Context, groupID primitive.ObjectID, member domain.GroupMember) error {
	filter := bson.M{"_id": groupID, "isDeleted": false}
	update := bson.M{"$pull": bson.M{"members": bson.M{"memberId": member.MemberId}}}

	res, err := gr.Collection.UpdateOne(context.TODO(), filter, update)
	if res.ModifiedCount < 1 {
		return errors.New("unable to remove member")
	}
	return err
}

func (gr GroupRepo) UpdateMemberRole(ctx context.Context, groupID primitive.ObjectID, member domain.GroupMember) error {
	filter := bson.M{"_id": groupID, "members.memberId": member.MemberId, "isDeleted": false}
	update := bson.M{"$set": bson.M{"members.$.role": member.Role}}

	res, err := gr.Collection.UpdateOne(context.TODO(), filter, update)
	if res.ModifiedCount < 1 {
		return errors.New("nothing to update")
	}

	return err
}

func (gr GroupRepo) Delete(ctx context.Context, groupID primitive.ObjectID) error {
	filter := bson.M{"_id": groupID}
	update := bson.M{"$set": bson.M{"isDeleted": true}}

	res, err := gr.Collection.UpdateOne(context.TODO(), filter, update)
	if res.ModifiedCount < 1 {
		return errors.New("nothing to update")
	}

	return err
}

func (gr GroupRepo) Archive(ctx context.Context, groupID primitive.ObjectID, status bool) error {
	filter := bson.M{"_id": groupID, "isDeleted": false}
	update := bson.M{"$set": bson.M{"isArchive": status}}

	res, err := gr.Collection.UpdateOne(context.TODO(), filter, update)
	if res.ModifiedCount < 1 {
		return errors.New("nothing to update")
	}

	return err
}

func (gr GroupRepo) GetGroupById(ctx context.Context, groupID primitive.ObjectID) (domain.Group, error) {
	filter := bson.M{"_id": groupID, "isDeleted": false}
	var group domain.Group
	err := gr.Collection.FindOne(ctx, filter).Decode(&group)
	if err != nil {
		return group, err
	}
	return group, nil
}

func (gr GroupRepo) GetGroupMemberRoleById(ctx context.Context, groupID primitive.ObjectID, memberID int) (domain.GroupMember, error) {
	var group domain.Group
	filter := bson.M{"_id": groupID, "members.memberId": memberID}
	projection := bson.M{"members.$": 1}

	err := gr.Collection.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&group)
	if err != nil {
		return domain.GroupMember{}, err
	}

	// Check if the group has been found
	if len(group.Members) == 0 {
		return domain.GroupMember{}, fmt.Errorf("member not found in group")
	}

	return group.Members[0], nil
}

func (gr GroupRepo) GetGroupsByBoardId(ctx context.Context, boardId primitive.ObjectID, memberId int) ([]domain.Group, error) {
	var groups []domain.Group
	filter := bson.M{"boardId": boardId, "isDeleted": false}

	cursor, err := gr.Collection.Find(ctx, filter)
	if err != nil {
		return groups, err
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &groups)
	if err != nil {
		return groups, err
	}

	return groups, nil
}
