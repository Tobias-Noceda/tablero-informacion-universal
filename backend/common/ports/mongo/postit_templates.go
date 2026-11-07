package mongo

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ infrastructure.PostitTemplateStore = (*MongoDB)(nil)

func (db *MongoDB) ensurePostitTemplateIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.postitTemplates.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "scope.kind", Value: 1}, {Key: "scope.owner", Value: 1}, {Key: "name", Value: 1}},
	})
	return err
}

func (db *MongoDB) FindPostitTemplate(key string) (*models.PostitTemplate, error) {
	ctx, cancel := timeout()
	defer cancel()

	c := &models.PostitTemplate{}
	if err := db.postitTemplates.FindOne(ctx, bson.M{"_id": key}).Decode(c); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrPostitTemplateNotFound
		}
		return nil, err
	}
	return c, nil
}

func (db *MongoDB) FindPostitTemplates(keys []string) ([]models.PostitTemplate, error) {
	if len(keys) == 0 {
		return []models.PostitTemplate{}, nil
	}
	return db.findPostitTemplates(bson.M{"_id": bson.M{"$in": keys}})
}

func (db *MongoDB) ListPostitTemplates(scopes ...models.PostitTemplateScope) ([]models.PostitTemplate, error) {
	if len(scopes) == 0 {
		return []models.PostitTemplate{}, nil
	}

	or := make(bson.A, 0, len(scopes))
	for _, scope := range scopes {
		or = append(or, bson.M{"scope.kind": scope.Kind, "scope.owner": scope.Owner})
	}
	return db.findPostitTemplates(bson.M{"$or": or})
}

func (db *MongoDB) findPostitTemplates(filter bson.M) ([]models.PostitTemplate, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.postitTemplates.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}

	list := []models.PostitTemplate{}
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (db *MongoDB) CountPostitTemplates(scope models.PostitTemplateScope) (int64, error) {
	ctx, cancel := timeout()
	defer cancel()

	return db.postitTemplates.CountDocuments(ctx, bson.M{"scope.kind": scope.Kind, "scope.owner": scope.Owner})
}

func (db *MongoDB) PutPostitTemplate(c *models.PostitTemplate) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.postitTemplates.ReplaceOne(ctx, bson.M{"_id": c.Key}, c, options.Replace().SetUpsert(true))
	return err
}

func (db *MongoDB) DeletePostitTemplate(key string) error {
	ctx, cancel := timeout()
	defer cancel()

	result, err := db.postitTemplates.DeleteOne(ctx, bson.M{"_id": key})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return infrastructure.ErrPostitTemplateNotFound
	}
	return nil
}

// purgePostitTemplates removes every card type of scope; cards already made
// from them keep their own copy.
func (db *MongoDB) purgePostitTemplates(scope models.PostitTemplateScope) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.postitTemplates.DeleteMany(ctx, bson.M{"scope.kind": scope.Kind, "scope.owner": scope.Owner})
	return err
}
