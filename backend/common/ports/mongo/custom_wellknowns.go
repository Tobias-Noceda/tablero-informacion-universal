package mongo

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ infrastructure.CustomWellKnownStore = (*MongoDB)(nil)

func (db *MongoDB) ensureCustomWellKnownIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.customWellKnowns.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "scope.kind", Value: 1}, {Key: "scope.owner", Value: 1}, {Key: "name", Value: 1}},
	})
	return err
}

func (db *MongoDB) FindCustomWellKnown(key string) (*models.CustomWellKnown, error) {
	ctx, cancel := timeout()
	defer cancel()

	c := &models.CustomWellKnown{}
	if err := db.customWellKnowns.FindOne(ctx, bson.M{"_id": key}).Decode(c); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrCustomWellKnownNotFound
		}
		return nil, err
	}
	return c, nil
}

func (db *MongoDB) FindCustomWellKnowns(keys []string) ([]models.CustomWellKnown, error) {
	if len(keys) == 0 {
		return []models.CustomWellKnown{}, nil
	}
	return db.findCustomWellKnowns(bson.M{"_id": bson.M{"$in": keys}})
}

func (db *MongoDB) ListCustomWellKnowns(scopes ...models.CustomWellKnownScope) ([]models.CustomWellKnown, error) {
	if len(scopes) == 0 {
		return []models.CustomWellKnown{}, nil
	}

	or := make(bson.A, 0, len(scopes))
	for _, scope := range scopes {
		or = append(or, bson.M{"scope.kind": scope.Kind, "scope.owner": scope.Owner})
	}
	return db.findCustomWellKnowns(bson.M{"$or": or})
}

func (db *MongoDB) findCustomWellKnowns(filter bson.M) ([]models.CustomWellKnown, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.customWellKnowns.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}

	list := []models.CustomWellKnown{}
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (db *MongoDB) CountCustomWellKnowns(scope models.CustomWellKnownScope) (int64, error) {
	ctx, cancel := timeout()
	defer cancel()

	return db.customWellKnowns.CountDocuments(ctx, bson.M{"scope.kind": scope.Kind, "scope.owner": scope.Owner})
}

func (db *MongoDB) PutCustomWellKnown(c *models.CustomWellKnown) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.customWellKnowns.ReplaceOne(ctx, bson.M{"_id": c.Key}, c, options.Replace().SetUpsert(true))
	return err
}

func (db *MongoDB) DeleteCustomWellKnown(key string) error {
	ctx, cancel := timeout()
	defer cancel()

	result, err := db.customWellKnowns.DeleteOne(ctx, bson.M{"_id": key})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return infrastructure.ErrCustomWellKnownNotFound
	}
	return nil
}

// purgeCustomWellKnowns removes every card type of scope; cards already made
// from them keep their own copy.
func (db *MongoDB) purgeCustomWellKnowns(scope models.CustomWellKnownScope) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.customWellKnowns.DeleteMany(ctx, bson.M{"scope.kind": scope.Kind, "scope.owner": scope.Owner})
	return err
}
