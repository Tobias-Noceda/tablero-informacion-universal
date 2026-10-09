package mongo

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ infrastructure.OrgStore = (*MongoDB)(nil)

func (db *MongoDB) ensureOrgIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.orgs.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "members.user", Value: 1}}})
	return err
}

func (db *MongoDB) CreateOrg(org *models.Org) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.orgs.InsertOne(ctx, org)
	return err
}

func (db *MongoDB) FindOrg(id uuid.UUID) (*models.Org, error) {
	ctx, cancel := timeout()
	defer cancel()

	org := &models.Org{}
	if err := db.orgs.FindOne(ctx, bson.M{"_id": id}).Decode(org); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, infrastructure.ErrOrgNotFound
		}
		return nil, err
	}

	return org, nil
}

func (db *MongoDB) FindUserOrgs(userID string) ([]models.Org, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.orgs.Find(ctx, bson.M{"members.user": userID})
	if err != nil {
		return nil, err
	}

	orgs := []models.Org{}
	if err := cursor.All(ctx, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}

func (db *MongoDB) RenameOrg(id uuid.UUID, name string) error {
	ctx, cancel := timeout()
	defer cancel()

	result, err := db.orgs.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"name": name}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return infrastructure.ErrOrgNotFound
	}

	return nil
}

// keepsAnAdmin matches an organization that is left with an admin once user
// stops being one: user is not an admin, or someone else also is.
func keepsAnAdmin(user string) bson.A {
	return bson.A{
		bson.M{"members": bson.M{"$not": bson.M{"$elemMatch": bson.M{"user": user, "role": models.OrgRoleAdmin}}}},
		bson.M{"members": bson.M{"$elemMatch": bson.M{"role": models.OrgRoleAdmin, "user": bson.M{"$ne": user}}}},
	}
}

// SetOrgMember changes the member's role in place, or appends them. Both
// writes carry their condition in the filter, so concurrent calls can neither
// list someone twice nor demote the last admin.
func (db *MongoDB) SetOrgMember(id uuid.UUID, userID string, role models.OrgRole) error {
	ctx, cancel := timeout()
	defer cancel()

	filter := bson.M{"_id": id, "members.user": userID}
	if role != models.OrgRoleAdmin {
		filter["$or"] = keepsAnAdmin(userID)
	}
	result, err := db.orgs.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"members.$[m].role": role}},
		options.UpdateOne().SetArrayFilters([]any{bson.M{"m.user": userID}}),
	)
	if err != nil {
		return err
	}
	if result.MatchedCount > 0 {
		return nil
	}

	result, err = db.orgs.UpdateOne(ctx,
		bson.M{"_id": id, "members.user": bson.M{"$ne": userID}},
		bson.M{"$push": bson.M{"members": models.OrgMember{User: userID, Role: role}}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount > 0 {
		return nil
	}

	// Neither matched: the org is gone, the user is its last admin, or they
	// were added in between (then try again).
	org, err := db.FindOrg(id)
	if err != nil {
		return err
	}
	if role != models.OrgRoleAdmin && org.RoleOf(userID) == models.OrgRoleAdmin && org.Admins() == 1 {
		return infrastructure.ErrLastOrgAdmin
	}
	return db.SetOrgMember(id, userID, role)
}

func (db *MongoDB) RemoveOrgMember(id uuid.UUID, userID string) error {
	ctx, cancel := timeout()
	defer cancel()

	result, err := db.orgs.UpdateOne(ctx,
		bson.M{"_id": id, "$or": keepsAnAdmin(userID)},
		bson.M{"$pull": bson.M{"members": bson.M{"user": userID}}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount > 0 {
		return nil
	}

	if _, err := db.FindOrg(id); err != nil {
		return err
	}
	return infrastructure.ErrLastOrgAdmin
}

func (db *MongoDB) DeleteOrg(id uuid.UUID) error {
	ctx, cancel := timeout()
	defer cancel()

	result, err := db.orgs.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return infrastructure.ErrOrgNotFound
	}

	return nil
}

func (db *MongoDB) CountOrgBoards(id uuid.UUID) (int64, error) {
	ctx, cancel := timeout()
	defer cancel()

	return db.boards.CountDocuments(ctx, bson.M{"org": id})
}

func (db *MongoDB) FindOrgBoards(id uuid.UUID) ([]models.Board, error) {
	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.boards.Find(ctx, bson.M{"org": id})
	if err != nil {
		return nil, err
	}

	boards := []models.Board{}
	if err := cursor.All(ctx, &boards); err != nil {
		return nil, err
	}
	return boards, nil
}

func (db *MongoDB) CountOrgGroups(id uuid.UUID) (int64, error) {
	ctx, cancel := timeout()
	defer cancel()

	return db.groups.CountDocuments(ctx, bson.M{"org": id})
}

// reachedBy matches what one of listed conditions names, or what belongs to
// one of orgs.
func reachedBy(listed bson.A, orgs []uuid.UUID) bson.M {
	if len(orgs) > 0 {
		listed = append(listed, bson.M{"org": bson.M{"$in": orgs}})
	}
	return bson.M{"$or": listed}
}
