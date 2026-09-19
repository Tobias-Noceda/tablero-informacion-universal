package mongo

import (
	"github.com/Secreto31126/tesis/common/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func scopeFilter(scope models.SecretScope) bson.M {
	return bson.M{"scope.kind": scope.Kind, "scope.owner": scope.Owner}
}

func (db *MongoDB) ensureSecretIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.secrets.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "scope.kind", Value: 1}, {Key: "scope.owner", Value: 1}, {Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	return err
}

func (db *MongoDB) UpsertSecret(secret *models.Secret) error {
	ctx, cancel := timeout()
	defer cancel()

	filter := scopeFilter(secret.Scope)
	filter["name"] = secret.Name

	_, err := db.secrets.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"kind":       secret.Kind,
			"ciphertext": secret.Ciphertext,
			"nonce":      secret.Nonce,
			"keyversion": secret.KeyVersion,
			"updatedat":  secret.UpdatedAt,
			"flow":       secret.Flow,
			"authorized": secret.Authorized,
		},
		"$setOnInsert": bson.M{
			"_id":       secret.Id,
			"scope":     secret.Scope,
			"name":      secret.Name,
			"createdat": secret.CreatedAt,
		},
	}, options.UpdateOne().SetUpsert(true))

	return err
}

func (db *MongoDB) FindSecrets(scope models.SecretScope, names []string) ([]models.Secret, error) {
	ctx, cancel := timeout()
	defer cancel()

	filter := scopeFilter(scope)
	filter["name"] = bson.M{"$in": names}

	cursor, err := db.secrets.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var secrets []models.Secret
	if err := cursor.All(ctx, &secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}

func (db *MongoDB) ListSecrets(scope models.SecretScope) ([]models.Secret, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.secrets.Find(ctx, scopeFilter(scope))
	if err != nil {
		return nil, err
	}

	var secrets []models.Secret
	if err := cursor.All(ctx, &secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}

func (db *MongoDB) DeleteSecret(scope models.SecretScope, name string) error {
	ctx, cancel := timeout()
	defer cancel()

	filter := scopeFilter(scope)
	filter["name"] = name

	_, err := db.secrets.DeleteOne(ctx, filter)
	return err
}
