package mongo

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ infrastructure.GroupStore = (*MongoDB)(nil)

func (db *MongoDB) ensureGroupIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.groups.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "owner", Value: 1}}},
		{Keys: bson.D{{Key: "members", Value: 1}}},
	})

	return err
}

func (db *MongoDB) CreateGroup(group *models.Group) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.groups.InsertOne(ctx, group)
	return err
}

func (db *MongoDB) FindGroup(id uuid.UUID) (*models.Group, error) {
	ctx, cancel := timeout()
	defer cancel()

	group := &models.Group{}
	if err := db.groups.FindOne(ctx, bson.M{"_id": id}).Decode(group); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrGroupNotFound
		}
		return nil, err
	}

	return group, nil
}

func (db *MongoDB) FindUserGroups(userID string) ([]models.Group, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.groups.Find(ctx, bson.M{"$or": []bson.M{{"owner": userID}, {"members": userID}}})
	if err != nil {
		return nil, err
	}

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

func (db *MongoDB) DeleteGroup(id uuid.UUID) error {
	ctx, cancel := timeout()
	defer cancel()

	result, err := db.groups.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return infrastructure.ErrGroupNotFound
	}

	return nil
}

func (db *MongoDB) AddGroupMember(id uuid.UUID, userID string) error {
	return db.updateGroup(id, bson.M{"$addToSet": bson.M{"members": userID}})
}

func (db *MongoDB) RemoveGroupMember(id uuid.UUID, userID string) error {
	return db.updateGroup(id, bson.M{"$pull": bson.M{"members": userID}})
}

func (db *MongoDB) updateGroup(id uuid.UUID, update bson.M) error {
	ctx, cancel := timeout()
	defer cancel()

	result, err := db.groups.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return infrastructure.ErrGroupNotFound
	}

	return nil
}
