package mongo

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ infrastructure.KeyStore = (*MongoDB)(nil)

// One active key per scope, enforced by the database so concurrent
// provisioning of a scope's first key cannot produce two.
func (db *MongoDB) ensureDataKeyIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.dataKeys.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "scope.kind", Value: 1}, {Key: "scope.owner", Value: 1}},
		Options: options.Index().
			SetUnique(true).
			SetPartialFilterExpression(bson.M{"active": true}),
	})

	return err
}

func (db *MongoDB) FindActiveKey(scope models.SecretScope) (*models.DataKey, error) {
	ctx, cancel := timeout()
	defer cancel()

	filter := scopeFilter(scope)
	filter["active"] = true

	key := &models.DataKey{}
	if err := db.dataKeys.FindOne(ctx, filter).Decode(key); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrKeyNotFound
		}
		return nil, err
	}

	return key, nil
}

func (db *MongoDB) FindKey(id uuid.UUID) (*models.DataKey, error) {
	ctx, cancel := timeout()
	defer cancel()

	key := &models.DataKey{}
	if err := db.dataKeys.FindOne(ctx, bson.M{"_id": id}).Decode(key); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrKeyNotFound
		}
		return nil, err
	}

	return key, nil
}

func (db *MongoDB) InsertKey(key *models.DataKey) error {
	ctx, cancel := timeout()
	defer cancel()

	if _, err := db.dataKeys.InsertOne(ctx, key); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return infrastructure.ErrActiveKeyExists
		}
		return err
	}

	return nil
}

func (db *MongoDB) RetireKey(id uuid.UUID, at time.Time) error {
	ctx, cancel := timeout()
	defer cancel()

	res, err := db.dataKeys.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{"active": false, "retiredat": at},
	})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return infrastructure.ErrKeyNotFound
	}

	return nil
}

func (db *MongoDB) Rewrap(id uuid.UUID, wrapped, nonce []byte, kekVersion int) error {
	ctx, cancel := timeout()
	defer cancel()

	res, err := db.dataKeys.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{"wrappedkey": wrapped, "nonce": nonce, "kekversion": kekVersion},
	})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return infrastructure.ErrKeyNotFound
	}

	return nil
}

func (db *MongoDB) DeleteRetiredKeys(scope models.SecretScope) error {
	ctx, cancel := timeout()
	defer cancel()

	filter := scopeFilter(scope)
	filter["active"] = false

	_, err := db.dataKeys.DeleteMany(ctx, filter)
	return err
}

func (db *MongoDB) DeleteKeys(scope models.SecretScope) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.dataKeys.DeleteMany(ctx, scopeFilter(scope))
	return err
}

func (db *MongoDB) ListKeys() ([]models.DataKey, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.dataKeys.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	var keys []models.DataKey
	if err := cursor.All(ctx, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}
