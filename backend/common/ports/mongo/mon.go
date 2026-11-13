package mongo

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	MONGO_URL       = "mongodb://user:password@mongo:27017/?authSource=admin"
	MONGO_DATABASE  = "prod"
	REQUEST_TIMEOUT = 10 * time.Second
)

type MongoDB struct {
	client   *mongo.Client
	users    *mongo.Collection
	boards   *mongo.Collection
	postit   *mongo.Collection
	secrets  *mongo.Collection
	dataKeys *mongo.Collection
	groups   *mongo.Collection
	orgs     *mongo.Collection

	wellKnowns      *mongo.Collection
	postitTemplates *mongo.Collection

	accessTokens *mongo.Collection
}

func New() (*MongoDB, error) {
	db := &MongoDB{}
	reg := bson.NewRegistry()
	reg.RegisterTypeEncoder(uuidType, bson.ValueEncoderFunc(uuidEncodeValue))
	reg.RegisterTypeDecoder(uuidType, bson.ValueDecoderFunc(uuidDecodeValue))

	url := os.Getenv("MONGODB_URI")
	if url == "" {
		url = MONGO_URL
	}

	name := os.Getenv("MONGO_DATABASE")
	if name == "" {
		name = MONGO_DATABASE
	}

	clientOptions := options.Client().
		ApplyURI(url).
		SetRegistry(reg).
		SetBSONOptions(&options.BSONOptions{
			NilSliceAsEmpty: true,
			NilMapAsEmpty:   true,
		}).
		SetMaxConnIdleTime(30 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	db.client = client
	db.users = db.client.Database(name).Collection("users")
	db.boards = db.client.Database(name).Collection("boards")
	db.postit = db.client.Database(name).Collection("postit")
	db.secrets = db.client.Database(name).Collection("secrets")
	db.dataKeys = db.client.Database(name).Collection("data_keys")
	db.groups = db.client.Database(name).Collection("groups")
	db.orgs = db.client.Database(name).Collection("orgs")
	db.accessTokens = db.client.Database(name).Collection("access_tokens")
	db.wellKnowns = db.client.Database(name).Collection("well_knowns")
	db.postitTemplates = db.client.Database(name).Collection("postit_templates")

	return db, nil
}

func (db *MongoDB) Close() error {
	return db.client.Disconnect(context.Background())
}

// EnsureIndexes creates the indexes the stores rely on for uniqueness. Safe
// to call on every boot: Mongo ignores an index that already exists.
func (db *MongoDB) EnsureIndexes() error {
	if err := db.ensureSecretIndexes(); err != nil {
		return err
	}

	if err := db.ensureDataKeyIndexes(); err != nil {
		return err
	}

	if err := db.ensureGroupIndexes(); err != nil {
		return err
	}

	if err := db.ensureBoardIndexes(); err != nil {
		return err
	}

	if err := db.ensureOrgIndexes(); err != nil {
		return err
	}

	if err := db.ensureAccessTokenIndexes(); err != nil {
		return err
	}

	if err := db.ensurePostitTemplateIndexes(); err != nil {
		return err
	}

	return db.ensureUserIndexes()
}

func timeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
}

// FindUserBoards lists every board the user belongs to, owned, shared or
// through one of orgs.
func (db *MongoDB) FindUserBoards(user string, orgs []uuid.UUID) ([]models.Board, error) {
	ctx, cancel := timeout()
	defer cancel()

	boards := []models.Board{}

	filter := reachedBy(bson.A{bson.M{"owner": user}, bson.M{"members.user": user}}, orgs)

	cursor, err := db.boards.Find(ctx, filter)

	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &boards)
	if err != nil {
		return nil, err
	}

	return boards, nil
}

func (db *MongoDB) FindBoardPostIts(board_id uuid.UUID) ([]models.PostIts, error) {
	ctx, cancel := timeout()
	defer cancel()

	board := &models.Board{}

	filter := bson.M{"_id": board_id}

	err := db.boards.FindOne(ctx, filter).Decode(board)
	if err != nil {
		return nil, err
	}

	postItIDs := make([]uuid.UUID, 0, len(board.PostIts))
	for _, postIt := range board.PostIts {
		postItIDs = append(postItIDs, postIt.Id)
	}

	cursor, err := db.postit.Find(ctx, bson.M{
		"_id": bson.M{"$in": postItIDs},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	postIts := []models.PostIts{}

	if err := cursor.All(ctx, &postIts); err != nil {
		return nil, err
	}

	return postIts, nil
}

func (db *MongoDB) FindPostIt(id uuid.UUID) (*models.PostIts, error) {
	ctx, cancel := timeout()
	defer cancel()

	postIt := &models.PostIts{}

	filter := bson.M{"_id": id}

	err := db.postit.FindOne(ctx, filter).Decode(postIt)
	if err != nil {
		return nil, err
	}

	return postIt, nil
}

func (db *MongoDB) FindBoard(id uuid.UUID) (*models.Board, error) {
	ctx, cancel := timeout()
	defer cancel()

	board := &models.Board{}

	filter := bson.M{"_id": id}

	err := db.boards.FindOne(ctx, filter).Decode(board)
	if err != nil {
		return nil, err
	}

	return board, nil
}

func (db *MongoDB) DeletePostIt(id uuid.UUID) (strands []models.Strand, err error) {
	ctx, cancel := timeout()
	defer cancel()

	postIt := &models.PostIts{}
	if err := db.postit.FindOneAndDelete(ctx, bson.M{"_id": id}).Decode(postIt); err != nil {
		return nil, err
	}

	board := &models.Board{}
	if err := db.boards.FindOneAndUpdate(
		ctx,
		bson.M{"_id": postIt.Board},
		bson.M{
			"$pull": bson.M{
				"postits": bson.M{"id": id},
				"strands": bson.M{
					"$or": []bson.M{
						{"source": id},
						{"target": id},
					},
				},
			},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.Before),
	).Decode(board); err != nil {
		return nil, err
	}

	for _, s := range board.Strands {
		if s.Source == id || s.Target == id {
			strands = append(strands, s)
		}
	}

	return strands, nil
}

func (db *MongoDB) DeleteBoard(id uuid.UUID) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.postit.DeleteMany(ctx, bson.M{"board": id})
	if err != nil {
		return err
	}

	if _, err = db.boards.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		return err
	}

	return db.purgePostitTemplates(models.PostitTemplateScope{Kind: models.TemplateScopeBoard, Owner: id.String()})
}

func (db *MongoDB) UpdatePostIt(id uuid.UUID, set map[string]any) error {
	ctx, cancel := timeout()
	defer cancel()

	res, err := db.postit.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": set})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// Aux method added to the db
func (db *MongoDB) updateBoard(filter, update bson.M) error {
	ctx, cancel := timeout()
	defer cancel()

	res, err := db.boards.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (db *MongoDB) UpdateBoardName(id uuid.UUID, name string) error {
	return db.updateBoard(
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"name": name}},
	)
}

func (db *MongoDB) CreateBoard(name string, owner string, org *uuid.UUID) (*models.Board, error) {
	ctx, cancel := timeout()
	defer cancel()

	board := &models.Board{Id: uuid.New(), Name: name, Owner: owner, Org: org}

	_, err := db.boards.InsertOne(ctx, board)
	if err != nil {
		return nil, err
	}

	return board, nil
}

func (db *MongoDB) CreatePostIt(postIt *models.PostIts, title models.BoardPostItTitle, ptype string, pos models.Position) (*models.PostIts, error) {
	ctx, cancel := timeout()
	defer cancel()

	postIt.Id = uuid.New()
	if _, err := db.postit.InsertOne(ctx, postIt); err != nil {
		return nil, err
	}

	entry := models.BoardPostIt{Id: postIt.Id, Title: title, Type: ptype, Position: pos}

	res, err := db.boards.UpdateOne(
		ctx,
		bson.M{"_id": postIt.Board},
		bson.M{"$push": bson.M{"postits": entry}},
	)
	if err != nil {
		return nil, err
	}

	if res.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return postIt, nil
}

func (db *MongoDB) MovePostIt(boardID, postItID uuid.UUID, pos models.Position) error {
	return db.updateBoard(
		bson.M{"_id": boardID, "postits.id": postItID},
		bson.M{"$set": bson.M{"postits.$.position": pos}},
	)
}

func (db *MongoDB) EditTitlePostIt(boardID, postItID uuid.UUID, title models.BoardPostItTitle) error {
	return db.updateBoard(
		bson.M{"_id": boardID, "postits.id": postItID},
		bson.M{"$set": bson.M{"postits.$.title": title}},
	)
}

func (db *MongoDB) ConnectPostIts(boardID uuid.UUID, strand models.Strand) (*models.Strand, error) {
	strand.Id = uuid.New()

	err := db.updateBoard(
		bson.M{
			"_id": boardID,
			"postits.id": bson.M{
				"$all": []uuid.UUID{strand.Source, strand.Target},
			},
		},
		bson.M{"$addToSet": bson.M{"strands": strand}},
	)
	if err != nil {
		return nil, err
	}

	return &strand, nil
}

// UpdateStrand sets the patch's fields on the strand and unsets the empty
// ones, so a default style is stored as no style at all.
func (db *MongoDB) UpdateStrand(boardID, strandID uuid.UUID, patch models.StrandPatch) (*models.Strand, error) {
	set, unset := bson.M{}, bson.M{}
	put := func(field string, value *string) {
		switch {
		case value == nil:
		case *value == "":
			unset["strands.$."+field] = ""
		default:
			set["strands.$."+field] = *value
		}
	}
	put("label", patch.Label)
	put("arrow", (*string)(patch.Arrow))
	put("width", (*string)(patch.Width))
	put("color", (*string)(patch.Color))

	update := bson.M{}
	if len(set) > 0 {
		update["$set"] = set
	}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	if len(update) == 0 {
		return nil, errors.New("update strand: the patch changes nothing")
	}

	ctx, cancel := timeout()
	defer cancel()

	board := &models.Board{}
	err := db.boards.FindOneAndUpdate(ctx,
		bson.M{"_id": boardID, "strands.id": strandID},
		update,
		options.FindOneAndUpdate().
			SetReturnDocument(options.After).
			SetProjection(bson.M{"strands": bson.M{"$elemMatch": bson.M{"id": strandID}}}),
	).Decode(board)
	if errors.Is(err, mongo.ErrNoDocuments) || (err == nil && len(board.Strands) == 0) {
		return nil, infrastructure.ErrStrandNotFound
	}
	if err != nil {
		return nil, err
	}
	return &board.Strands[0], nil
}

func (db *MongoDB) DisconnectPostIts(boardID, strandID uuid.UUID) error {
	return db.updateBoard(
		bson.M{"_id": boardID},
		bson.M{"$pull": bson.M{
			"strands": bson.M{
				"id": strandID,
			},
		}},
	)
}

func (db *MongoDB) ensureBoardIndexes() error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := db.boards.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "owner", Value: 1}}},
		{Keys: bson.D{{Key: "members.user", Value: 1}}},
		{Keys: bson.D{{Key: "org", Value: 1}}},
	})

	return err
}

// SetBoardMember changes the member's role in place, or appends them. The
// append only matches while the user is absent, so two concurrent calls can
// never list someone twice.
func (db *MongoDB) SetBoardMember(boardID uuid.UUID, user string, role models.BoardRole) error {
	ctx, cancel := timeout()
	defer cancel()

	res, err := db.boards.UpdateOne(ctx,
		bson.M{"_id": boardID, "members.user": user},
		bson.M{"$set": bson.M{"members.$.role": role}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount > 0 {
		return nil
	}

	res, err = db.boards.UpdateOne(ctx,
		bson.M{"_id": boardID, "members.user": bson.M{"$ne": user}},
		bson.M{"$push": bson.M{"members": models.BoardMember{User: user, Role: role}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		// Either the board is gone or someone added the user in between;
		// only the first is a failure.
		if _, err := db.FindBoard(boardID); err != nil {
			return err
		}
		return db.SetBoardMember(boardID, user, role)
	}
	return nil
}

func (db *MongoDB) RemoveBoardMember(boardID uuid.UUID, user string) error {
	return db.updateBoard(
		bson.M{"_id": boardID},
		bson.M{"$pull": bson.M{"members": bson.M{"user": user}}},
	)
}

var db, _ = New()
var _ infrastructure.Database = db
