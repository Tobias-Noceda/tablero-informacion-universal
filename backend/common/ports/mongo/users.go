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

// LEGACY_IDENTITY_INDEX indexed identities.provider + identities.subject.
// Its partial filter admits whole documents, so the subject-less password
// identity of every user with Google linked indexed (password, null) and the
// second such user collided with the first.
const LEGACY_IDENTITY_INDEX = "identities.provider_1_identities.subject_1"

const (
	NAMESPACE_NOT_FOUND = 26
	INDEX_NOT_FOUND     = 27
)

// userDocument stores the user next to the keys of its external identities:
// a top level array that holds only the identities with a subject, which is
// what keeps them unique across users.
type userDocument struct {
	models.User  `bson:",inline"`
	IdentityKeys []string `bson:"identitykeys,omitempty"`
}

func identityKey(provider models.IdentityProvider, subject string) string {
	return string(provider) + ":" + subject
}

func identityKeys(identities []models.Identity) []string {
	var keys []string
	for _, identity := range identities {
		if identity.Subject != "" {
			keys = append(keys, identityKey(identity.Provider, identity.Subject))
		}
	}
	return keys
}

func (db *MongoDB) ensureUserIndexes() error {
	if err := db.upgradeIdentityIndex(); err != nil {
		return err
	}

	ctx, cancel := timeout()
	defer cancel()

	_, err := db.users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "identitykeys", Value: 1}},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.M{"identitykeys": bson.M{"$exists": true}}),
		},
	})

	return err
}

// upgradeIdentityIndex drops LEGACY_IDENTITY_INDEX and fills identitykeys in
// the users stored before it existed.
func (db *MongoDB) upgradeIdentityIndex() error {
	ctx, cancel := timeout()
	defer cancel()

	if err := db.users.Indexes().DropOne(ctx, LEGACY_IDENTITY_INDEX); err != nil {
		var command mongo.CommandError
		if !errors.As(err, &command) || (command.Code != NAMESPACE_NOT_FOUND && command.Code != INDEX_NOT_FOUND) {
			return err
		}
	}

	cursor, err := db.users.Find(ctx, bson.M{
		"identities.subject": bson.M{"$exists": true},
		"identitykeys":       bson.M{"$exists": false},
	})
	if err != nil {
		return err
	}

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return err
	}

	for _, user := range users {
		set := bson.M{"identitykeys": identityKeys(user.Identities)}
		if _, err := db.users.UpdateOne(ctx, bson.M{"_id": user.Id}, bson.M{"$set": set}); err != nil {
			return err
		}
	}

	return nil
}

func (db *MongoDB) CreateUser(user *models.User) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.users.InsertOne(ctx, userDocument{User: *user, IdentityKeys: identityKeys(user.Identities)})
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
	return db.findUser(bson.M{"identitykeys": identityKey(provider, subject)})
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
	ctx, cancel := timeout()
	defer cancel()

	set := bson.M{"identities": identities, "updatedat": time.Now().UTC()}
	update := bson.M{"$set": set}
	if keys := identityKeys(identities); len(keys) > 0 {
		set["identitykeys"] = keys
	} else {
		update["$unset"] = bson.M{"identitykeys": ""}
	}

	res, err := db.users.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return infrastructure.ErrUserNotFound
	}

	return nil
}
