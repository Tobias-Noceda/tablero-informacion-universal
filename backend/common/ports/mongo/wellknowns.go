package mongo

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ infrastructure.WellKnownStore = (*MongoDB)(nil)

func (db *MongoDB) FindWellKnown(key string) (*models.WellKnown, error) {
	ctx, cancel := timeout()
	defer cancel()

	wk := &models.WellKnown{}
	if err := db.wellKnowns.FindOne(ctx, bson.M{"_id": key}).Decode(wk); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrWellKnownNotFound
		}
		return nil, err
	}

	return wk, nil
}

func (db *MongoDB) ListWellKnowns() ([]models.WellKnown, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.wellKnowns.Find(ctx, bson.M{"deleted": bson.M{"$ne": true}}, options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}

	wks := []models.WellKnown{}
	if err := cursor.All(ctx, &wks); err != nil {
		return nil, err
	}

	return wks, nil
}

func (db *MongoDB) PutWellKnown(wk *models.WellKnown) error {
	ctx, cancel := timeout()
	defer cancel()

	stored := *wk
	stored.Deleted = false

	_, err := db.wellKnowns.ReplaceOne(ctx, bson.M{"_id": wk.Key}, stored, options.Replace().SetUpsert(true))
	return err
}

func (db *MongoDB) DeleteWellKnown(key string) error {
	return db.setDeleted(key, true)
}

func (db *MongoDB) RestoreWellKnown(key string) error {
	err := db.setDeleted(key, false)
	if !errors.Is(err, infrastructure.ErrWellKnownNotFound) {
		return err
	}

	if _, err = db.FindWellKnown(key); err != nil {
		return err
	}
	return infrastructure.ErrWellKnownNotDeleted
}

// setDeleted flips the tombstone, matching only a document in the other state.
func (db *MongoDB) setDeleted(key string, deleted bool) error {
	ctx, cancel := timeout()
	defer cancel()

	filter := bson.M{"_id": key, "deleted": bson.M{"$ne": deleted}}
	result, err := db.wellKnowns.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"deleted": deleted}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return infrastructure.ErrWellKnownNotFound
	}

	return nil
}

// typedParameters gives a stored definition's date and date-time parameters
// the type its built-in now declares. They were seeded as text before those
// types existed, and seeding never rewrites a stored definition; only a
// parameter still typed as text changes, so a type an admin chose stays.
func typedParameters(wk models.WellKnown) []mongo.WriteModel {
	var writes []mongo.WriteModel
	for _, p := range wk.Parameters {
		if p.Type != models.ParameterDate && p.Type != models.ParameterDateTime {
			continue
		}
		writes = append(writes, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": wk.Key}).
			SetUpdate(bson.M{"$set": bson.M{"parameters.$[p].type": p.Type}}).
			SetArrayFilters([]any{bson.M{"p.key": p.Key, "p.type": models.ParameterString}}))
	}
	return writes
}

func (db *MongoDB) SeedWellKnowns(defs []models.WellKnown) error {
	if len(defs) == 0 {
		return nil
	}

	ctx, cancel := timeout()
	defer cancel()

	writes := make([]mongo.WriteModel, 0, len(defs))
	for _, wk := range defs {
		writes = append(writes, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": wk.Key}).
			SetUpdate(bson.M{"$setOnInsert": wk}).
			SetUpsert(true))
		writes = append(writes, typedParameters(wk)...)
	}

	_, err := db.wellKnowns.BulkWrite(ctx, writes, options.BulkWrite().SetOrdered(false))
	return err
}
