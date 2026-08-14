package mongo

import (
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (db *MongoDB) UpsertSecret(secret *models.Secret) error {
	ctx, cancel := timeout()
	defer cancel()

	filter := bson.M{"board": secret.Board, "name": secret.Name}

	_, err := db.secrets.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"kind":       secret.Kind,
			"ciphertext": secret.Ciphertext,
			"nonce":      secret.Nonce,
			"keyversion": secret.KeyVersion,
			"updatedat":  secret.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"_id":       secret.Id,
			"board":     secret.Board,
			"name":      secret.Name,
			"createdat": secret.CreatedAt,
		},
	}, options.UpdateOne().SetUpsert(true))

	return err
}

func (db *MongoDB) FindSecrets(board uuid.UUID, names []string) ([]models.Secret, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.secrets.Find(ctx, bson.M{
		"board": board,
		"name":  bson.M{"$in": names},
	})
	if err != nil {
		return nil, err
	}

	var secrets []models.Secret
	if err := cursor.All(ctx, &secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}

func (db *MongoDB) ListSecrets(board uuid.UUID) ([]models.Secret, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.secrets.Find(ctx, bson.M{"board": board})
	if err != nil {
		return nil, err
	}

	var secrets []models.Secret
	if err := cursor.All(ctx, &secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}

func (db *MongoDB) DeleteSecret(board uuid.UUID, name string) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.secrets.DeleteOne(ctx, bson.M{"board": board, "name": name})
	return err
}
