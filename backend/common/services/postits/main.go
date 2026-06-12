package postits

import (
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type PostItsService struct {
	db    infrastructure.Database
	cache infrastructure.Cache
	run   infrastructure.Executer
}

func New(db infrastructure.Database, cache infrastructure.Cache, run infrastructure.Executer) *PostItsService {
	return &PostItsService{db, cache, run}
}

func (srv *PostItsService) CreatePostIt(postIt *models.PostIts) (*models.PostIts, error) {
	if postIt.WellKnown != "" {
		wk, err := findWellKnown(postIt.WellKnown, postIt.Params)
		if err != nil {
			return nil, err
		}

		wk.Board = postIt.Board
		postIt = wk
	}

	return srv.db.CreatePostIt(postIt, postIt.WellKnown, models.Position{X: 0, Y: 0})
}

func (srv *PostItsService) GetPostIt(id uuid.UUID) (*models.PostIts, error) {
	return srv.db.FindPostIt(id)
}

func (srv *PostItsService) UpdatePostIt(id uuid.UUID, set map[string]any) error {
	if len(set) == 0 {
		return nil
	}

	return srv.db.UpdatePostIt(id, set)
}

func (srv *PostItsService) MovePostIt(id uuid.UUID, pos models.Position) error {
	postit, err := srv.GetPostIt(id)
	if err != nil {
		return err
	}

	return srv.db.MovePostIt(postit.Board, postit.Id, pos)
}

func (srv *PostItsService) DeletePostIt(id uuid.UUID) error {
	return srv.db.DeletePostIt(id)
}

func (srv *PostItsService) ExecutePostIt(postit *models.PostIts) (any, error) {
	cached, err := srv.cache.FindPostItResult(postit.Id)
	if err == nil {
		return cached, nil
	}

	data, err := srv.run.Execute(postit)
	if err != nil {
		return nil, err
	}

	go srv.cache.AddPostItResult(postit, data)
	return data, nil
}
