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

var _ infrastructure.UserStore = (*MongoDB)(nil)

func (db *MongoDB) ensureUserIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "identities.provider", Value: 1}, {Key: "identities.subject", Value: 1}},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.M{"identities.subject": bson.M{"$exists": true}}),
		},
	})

	return err
}

func (db *MongoDB) CreateUser(user *models.User) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.users.InsertOne(ctx, user)
	if mongo.IsDuplicateKeyError(err) {
		return infrastructure.ErrEmailTaken
	}
	return err
}

func (db *MongoDB) FindUser(id uuid.UUID) (*models.User, error) {
	return db.findUser(bson.M{"_id": id})
}

func (db *MongoDB) FindUserByEmail(email string) (*models.User, error) {
	return db.findUser(bson.M{"email": email})
}

func (db *MongoDB) FindUserByIdentity(provider models.IdentityProvider, subject string) (*models.User, error) {
	return db.findUser(bson.M{"identities": bson.M{"$elemMatch": bson.M{"provider": provider, "subject": subject}}})
}

func (db *MongoDB) findUser(filter bson.M) (*models.User, error) {
	ctx, cancel := timeout()
	defer cancel()

	user := &models.User{}
	if err := db.users.FindOne(ctx, filter).Decode(user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (db *MongoDB) FindUsers(ids []string) ([]models.User, error) {
	ctx, cancel := timeout()
	defer cancel()

	parsed := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if u, err := uuid.Parse(id); err == nil {
			parsed = append(parsed, u)
		}
	}

	cursor, err := db.users.Find(ctx, bson.M{"_id": bson.M{"$in": parsed}})
	if err != nil {
		return nil, err
	}

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

func (db *MongoDB) UpdateUser(id uuid.UUID, set map[string]any) error {
	ctx, cancel := timeout()
	defer cancel()

	set["updatedat"] = time.Now().UTC()

	res, err := db.users.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return infrastructure.ErrUserNotFound
	}

	return nil
}

func (db *MongoDB) ReplaceIdentities(id uuid.UUID, identities []models.Identity) error {
	return db.UpdateUser(id, map[string]any{"identities": identities})
}
