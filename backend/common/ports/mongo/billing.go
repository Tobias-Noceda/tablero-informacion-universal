package mongo

import (
	"errors"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// BackfillBilling gives what was stored before plans existed someone to bill:
// an owner to every organization and a payer to every secret. It only writes
// where the field is absent, so a second run changes nothing. Organizations
// go first: a board's secrets are billed to its organization's owner.
func (db *MongoDB) BackfillBilling() error {
	if err := db.backfillOrgOwners(); err != nil {
		return err
	}
	return db.backfillSecretPayers()
}

func absent(field string) bson.M {
	return bson.M{field: bson.M{"$exists": false}}
}

// backfillOrgOwners makes the first admin the owner of every organization
// without one, which is who Org.Payer already answers for them.
func (db *MongoDB) backfillOrgOwners() error {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.orgs.Find(ctx, absent("owner"))
	if err != nil {
		return err
	}

	var orgs []models.Org
	if err := cursor.All(ctx, &orgs); err != nil {
		return err
	}

	for _, org := range orgs {
		owner := org.Payer()
		if owner == "" {
			continue
		}
		filter := absent("owner")
		filter["_id"] = org.Id
		if _, err := db.orgs.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"owner": owner}}); err != nil {
			return err
		}
	}

	return nil
}

// backfillSecretPayers stamps billed_to on the secrets without one, derived
// from the scope. The platform's own are never billed, and a secret whose
// board or group is gone is left as it is.
func (db *MongoDB) backfillSecretPayers() error {
	ctx, cancel := timeout()
	defer cancel()

	filter := absent("billed_to")
	filter["scope.kind"] = bson.M{"$ne": models.ScopeSystem}

	// Only the scope is needed; the ciphertexts stay where they are.
	cursor, err := db.secrets.Find(ctx, filter, options.Find().SetProjection(bson.M{"scope": 1}))
	if err != nil {
		return err
	}

	var secrets []struct {
		Id    uuid.UUID          `bson:"_id"`
		Scope models.SecretScope `bson:"scope"`
	}
	if err := cursor.All(ctx, &secrets); err != nil {
		return err
	}

	payers := map[models.SecretScope]string{}
	for _, secret := range secrets {
		payer, known := payers[secret.Scope]
		if !known {
			payer, err = db.scopePayer(secret.Scope)
			if err != nil {
				return err
			}
			payers[secret.Scope] = payer
		}
		if payer == "" {
			continue
		}

		filter := absent("billed_to")
		filter["_id"] = secret.Id
		if _, err := db.secrets.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"billed_to": payer}}); err != nil {
			return err
		}
	}

	return nil
}

// scopePayer is who a scope's secrets are billed to, "" for nobody: the
// user for their own and for what they keep on a board, the board's payer
// (its organization's, else its owner) for a board's, the owner for a
// group's.
func (db *MongoDB) scopePayer(scope models.SecretScope) (string, error) {
	switch scope.Kind {
	case models.ScopeUser:
		return scope.Owner, nil
	case models.ScopeMember:
		_, user, _ := scope.Member()
		return user, nil
	case models.ScopeBoard:
		id, err := uuid.Parse(scope.Owner)
		if err != nil {
			return "", nil
		}
		return db.boardPayer(id)
	case models.ScopeGroup:
		id, err := uuid.Parse(scope.Owner)
		if err != nil {
			return "", nil
		}
		ctx, cancel := timeout()
		defer cancel()

		var group models.Group
		if err := db.groups.FindOne(ctx, bson.M{"_id": id}).Decode(&group); err != nil {
			return "", ignoreMissing(err)
		}
		return group.Owner, nil
	}
	return "", nil
}

func (db *MongoDB) boardPayer(id uuid.UUID) (string, error) {
	ctx, cancel := timeout()
	defer cancel()

	var board models.Board
	if err := db.boards.FindOne(ctx, bson.M{"_id": id}).Decode(&board); err != nil {
		return "", ignoreMissing(err)
	}
	if board.Org == nil {
		return board.Owner, nil
	}

	var org models.Org
	if err := db.orgs.FindOne(ctx, bson.M{"_id": *board.Org}).Decode(&org); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return board.Owner, nil
		}
		return "", err
	}
	if payer := org.Payer(); payer != "" {
		return payer, nil
	}
	return board.Owner, nil
}

func ignoreMissing(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	return err
}
