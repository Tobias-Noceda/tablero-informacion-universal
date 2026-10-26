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

var _ infrastructure.AccessTokenStore = (*MongoDB)(nil)

// TOUCH_INTERVAL is how stale last_used_at may get before a use rewrites it.
const TOUCH_INTERVAL = time.Minute

func (db *MongoDB) ensureAccessTokenIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.accessTokens.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "hash", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user", Value: 1}}},
		// Mongo drops a token shortly after it expires; Resolve still checks
		// the date, since the TTL monitor only runs once a minute.
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})

	return err
}

func (db *MongoDB) CreateAccessToken(token *models.AccessToken) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.accessTokens.InsertOne(ctx, token)
	return err
}

func (db *MongoDB) FindAccessTokenByHash(hash string) (*models.AccessToken, error) {
	ctx, cancel := timeout()
	defer cancel()

	token := &models.AccessToken{}
	if err := db.accessTokens.FindOne(ctx, bson.M{"hash": hash}).Decode(token); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrAccessTokenNotFound
		}
		return nil, err
	}

	return token, nil
}

func (db *MongoDB) ListAccessTokens(user string) ([]models.AccessToken, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.accessTokens.Find(ctx, bson.M{"user": user}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}

	tokens := []models.AccessToken{}
	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (db *MongoDB) DeleteAccessToken(user string, id uuid.UUID) error {
	ctx, cancel := timeout()
	defer cancel()

	result, err := db.accessTokens.DeleteOne(ctx, bson.M{"_id": id, "user": user})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return infrastructure.ErrAccessTokenNotFound
	}

	return nil
}

func (db *MongoDB) TouchAccessToken(id uuid.UUID, at time.Time) error {
	ctx, cancel := timeout()
	defer cancel()

	// The staleness check lives in the filter, so a busy token costs one
	// write a minute instead of one per request.
	filter := bson.M{"_id": id, "$or": bson.A{
		bson.M{"last_used_at": bson.M{"$exists": false}},
		bson.M{"last_used_at": bson.M{"$lt": at.Add(-TOUCH_INTERVAL)}},
	}}
	_, err := db.accessTokens.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"last_used_at": at}})
	return err
}
