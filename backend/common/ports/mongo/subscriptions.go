package mongo

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (db *MongoDB) ensureSubscriptionIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.subscriptions.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user", Value: 1}, {Key: "created_at", Value: -1}},
	})
	return err
}

func (db *MongoDB) UpsertSubscription(sub *models.Subscription) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.subscriptions.ReplaceOne(ctx, bson.M{"_id": sub.Id}, sub, options.Replace().SetUpsert(true))
	return err
}

func (db *MongoDB) findSubscription(filter bson.M, opts ...options.Lister[options.FindOneOptions]) (*models.Subscription, error) {
	ctx, cancel := timeout()
	defer cancel()

	var sub models.Subscription
	if err := db.subscriptions.FindOne(ctx, filter, opts...).Decode(&sub); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (db *MongoDB) FindSubscription(id string) (*models.Subscription, error) {
	return db.findSubscription(bson.M{"_id": id})
}

func (db *MongoDB) LatestSubscription(user string) (*models.Subscription, error) {
	return db.findSubscription(bson.M{"user": user}, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}))
}

func (db *MongoDB) OpenSubscriptionOf(user string) (*models.Subscription, error) {
	return db.findSubscription(
		bson.M{"user": user, "status": bson.M{"$in": bson.A{models.SubscriptionAuthorized, models.SubscriptionPaused}}},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
}

func (db *MongoDB) OpenSubscriptions() ([]models.Subscription, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.subscriptions.Find(ctx, bson.M{"status": bson.M{"$ne": models.SubscriptionCanceled}})
	if err != nil {
		return nil, err
	}

	subs := []models.Subscription{}
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}
