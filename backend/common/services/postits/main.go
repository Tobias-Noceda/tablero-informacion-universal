package postits

import (
	"log"
	"maps"
	"strings"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type PostItsService struct {
	db      infrastructure.Database
	cache   infrastructure.Cache
	run     infrastructure.Executer
	secrets infrastructure.SecretResolver
}

func New(db infrastructure.Database, cache infrastructure.Cache, run infrastructure.Executer, secrets infrastructure.SecretResolver) *PostItsService {
	return &PostItsService{db, cache, run, secrets}
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

	return srv.db.CreatePostIt(postIt, postIt.WellKnown, models.Position{X: 750, Y: 350})
}

func (srv *PostItsService) GetPostIt(id uuid.UUID) (*models.PostIts, error) {
	return srv.db.FindPostIt(id)
}

func (srv *PostItsService) UpdatePostIt(id uuid.UUID, set map[string]any) error {
	if len(set) == 0 {
		return nil
	}

	if err := srv.db.UpdatePostIt(id, set); err != nil {
		return err
	}
	return srv.cache.DropPostItResult(id)
}

func (srv *PostItsService) MovePostIt(id uuid.UUID, pos models.Position) error {
	postit, err := srv.GetPostIt(id)
	if err != nil {
		return err
	}

	return srv.db.MovePostIt(postit.Board, postit.Id, pos)
}

func (srv *PostItsService) DeletePostIt(id uuid.UUID) ([]models.Strand, error) {
	return srv.db.DeletePostIt(id)
}

func secretRefs(postit *models.PostIts) []string {
	seen := make(map[string]struct{})

	for _, source := range []map[string]string{postit.Request.Headers, postit.Request.Queries, postit.Params} {
		for _, value := range source {
			name, found := strings.CutPrefix(value, "$")
			if !found || !models.ValidSecretName(name) {
				continue
			}

			seen[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}

	return names
}

func (srv *PostItsService) prepare(postit *models.PostIts) (*models.PostIts, error) {
	resolved, err := srv.secrets.Resolve(postit.Board, secretRefs(postit))
	if err != nil {
		return nil, err
	}

	clone := *postit
	clone.Request.Headers = maps.Clone(postit.Request.Headers)
	clone.Request.Queries = maps.Clone(postit.Request.Queries)

	clone.Params = maps.Clone(postit.Params)
	if clone.Params == nil {
		clone.Params = make(map[string]string, len(resolved))
	}

	for param, value := range clone.Params {
		if secret, ok := resolved[value]; ok {
			clone.Params[param] = secret
		}
	}

	maps.Copy(clone.Params, resolved)

	return &clone, nil
}

func (srv *PostItsService) ExecutePostIt(postit *models.PostIts) (any, error) {
	cached, err := srv.cache.FindPostItResult(postit.Id)
	if err == nil {
		return cached, nil
	}

	prepared := postit
	if postit.Resource != nil {
		prepared, err = srv.prepare(postit)
		if err != nil {
			return nil, err
		}
	}

	data, err := srv.run.Execute(prepared)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := srv.cache.AddPostItResult(postit, data); err != nil {
			log.Printf("post-it %s: caching result failed: %v", postit.Id, err)
		}
	}()

	return data, nil
}
