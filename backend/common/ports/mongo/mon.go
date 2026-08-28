package mongo

import (
	"context"
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
	client  *mongo.Client
	users   *mongo.Collection
	boards  *mongo.Collection
	postit  *mongo.Collection
	secrets *mongo.Collection
}

func New() (*MongoDB, error) {
	db := &MongoDB{}
	reg := bson.NewRegistry()
	reg.RegisterTypeEncoder(uuidType, bson.ValueEncoderFunc(uuidEncodeValue))
	reg.RegisterTypeDecoder(uuidType, bson.ValueDecoderFunc(uuidDecodeValue))

	url := os.Getenv("MONGO_URI")
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

	return db, nil
}

func (db *MongoDB) Close() error {
	return db.client.Disconnect(context.Background())
}

func timeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
}

func (db *MongoDB) FindUserBoards(cognito_id string) ([]models.Board, error) {
	ctx, cancel := timeout()
	defer cancel()

	var boards []models.Board

	filter := bson.M{"owner": cognito_id}

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

	_, err = db.boards.DeleteOne(ctx, bson.M{"_id": id})
	return err
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

func (db *MongoDB) CreateBoard(name string, owner string) (*models.Board, error) {
	ctx, cancel := timeout()
	defer cancel()

	board := &models.Board{Id: uuid.New(), Name: name, Owner: owner}

	_, err := db.boards.InsertOne(ctx, board)
	if err != nil {
		return nil, err
	}

	return board, nil
}

func (db *MongoDB) CreatePostIt(postIt *models.PostIts, ptype string, pos models.Position) (*models.PostIts, error) {
	ctx, cancel := timeout()
	defer cancel()

	postIt.Id = uuid.New()
	if _, err := db.postit.InsertOne(ctx, postIt); err != nil {
		return nil, err
	}

	entry := models.BoardPostIt{Id: postIt.Id, Type: ptype, Position: pos}

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

func (db *MongoDB) ConnectPostIts(boardID, source, target uuid.UUID) (*models.Strand, error) {
	strand := models.Strand{Id: uuid.New(), Source: source, Target: target}

	err := db.updateBoard(
		bson.M{
			"_id": boardID,
			"postits.id": bson.M{
				"$all": []uuid.UUID{source, target},
			},
		},
		bson.M{"$addToSet": bson.M{"strands": strand}},
	)
	if err != nil {
		return nil, err
	}

	return &strand, nil
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

func (db *MongoDB) AddCollaboratorToBoard(boardID uuid.UUID, cognitoID string) error {
	return db.updateBoard(
		bson.M{"_id": boardID},
		bson.M{"$addToSet": bson.M{"collaborators": cognitoID}},
	)
}

func (db *MongoDB) RemoveCollaboratorFromBoard(boardID uuid.UUID, cognitoID string) error {
	return db.updateBoard(
		bson.M{"_id": boardID},
		bson.M{"$pull": bson.M{"collaborators": cognitoID}},
	)
}

var db, _ = New()
var _ infrastructure.Database = db
