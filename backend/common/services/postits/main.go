package postits

import (
	"errors"
	"log"
	"maps"
	"strings"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	// ErrSystemSecretMissing is deliberately vague: which platform credential a
	// card needs is not the user's business, so the name only goes to the log.
	ErrSystemSecretMissing   = errors.New("This card is temporarily unavailable")
	ErrCredentialUnavailable = errors.New("This card's credentials are no longer available")
	// ErrNotAMember reads like a missing resource on purpose: to a stranger,
	// a board and its cards do not exist.
	ErrNotAMember     = errors.New("Not found")
	ErrInvalidBinding = errors.New("Binding aliases must match [A-Z][A-Z0-9_]*")
)

type PostItsService struct {
	db      infrastructure.Database
	cache   infrastructure.Cache
	run     infrastructure.Executer
	secrets infrastructure.SecretResolver
	access  infrastructure.Access
}

func New(db infrastructure.Database, cache infrastructure.Cache, run infrastructure.Executer, secrets infrastructure.SecretResolver, access infrastructure.Access) *PostItsService {
	return &PostItsService{db, cache, run, secrets, access}
}

func (srv *PostItsService) CreatePostIt(principal models.Principal, postIt *models.PostIts) (*models.PostIts, error) {
	if err := srv.require(principal, postIt.Board, models.BoardEditor); err != nil {
		return nil, err
	}

	if postIt.WellKnown != "" {
		wk, err := findWellKnown(postIt.WellKnown, postIt.Params)
		if err != nil {
			return nil, err
		}

		wk.Board = postIt.Board
		wk.Bindings = postIt.Bindings
		// An empty title keeps the well-known's default.
		if postIt.Title.Text != "" {
			wk.Title = postIt.Title
		}
		postIt = wk
	}

	if err := srv.bind(principal, postIt.Board, postIt.Bindings); err != nil {
		return nil, err
	}
	postIt.RunAs = principal.ID
	postIt.Title.Vars = titleVars(postIt.Title.Text, outputsOf(postIt))

	return srv.db.CreatePostIt(postIt, postIt.WellKnown, models.Position{X: 750, Y: 350})
}

// GetPostIt loads a card for anyone who may see its board. ExecutePostIt runs
// what it returns.
func (srv *PostItsService) GetPostIt(principal models.Principal, id uuid.UUID) (*models.PostIts, error) {
	return srv.load(principal, id, models.BoardViewer)
}

// load finds a card on whose board the principal holds at least min.
func (srv *PostItsService) load(principal models.Principal, id uuid.UUID, min models.BoardRole) (*models.PostIts, error) {
	postit, err := srv.db.FindPostIt(id)
	if err != nil || postit == nil {
		return nil, ErrNotAMember
	}

	if err := srv.require(principal, postit.Board, min); err != nil {
		return nil, err
	}

	return postit, nil
}

func (srv *PostItsService) UpdatePostIt(principal models.Principal, id uuid.UUID, set map[string]any) error {
	if len(set) == 0 {
		return nil
	}

	postit, err := srv.load(principal, id, models.BoardEditor)
	if err != nil {
		return err
	}

	if touchesCredentials(set) {
		bindings := postit.Bindings
		if edited, ok := set["bindings"].(map[string]models.SecretRef); ok {
			bindings = edited
		}
		if err := srv.bind(principal, postit.Board, bindings); err != nil {
			return err
		}
		set["runas"] = principal.ID
	}

	if title, ok := set["title"].(*models.Title); ok && title != nil {
		edited := *postit
		if params, ok := set["params"].(map[string]string); ok {
			edited.Params = params
		}
		if query, ok := set["query"].(map[string]string); ok {
			edited.Query = query
		}
		set["title"] = models.Title{Text: title.Text, Vars: titleVars(title.Text, outputsOf(&edited))}
	}

	if err := srv.db.UpdatePostIt(id, set); err != nil {
		return err
	}
	return srv.cache.DropPostItResult(id)
}

func touchesCredentials(set map[string]any) bool {
	_, params := set["params"]
	_, bindings := set["bindings"]
	return params || bindings
}

func (srv *PostItsService) require(principal models.Principal, boardID uuid.UUID, min models.BoardRole) error {
	board, err := srv.db.FindBoard(boardID)
	if err != nil || board == nil || !srv.access.BoardRole(principal, board).AtLeast(min) {
		return ErrNotAMember
	}
	return nil
}

// bind is the authorization event: whoever saves the card must be allowed to
// use every secret it names, right now.
func (srv *PostItsService) bind(principal models.Principal, boardID uuid.UUID, bindings map[string]models.SecretRef) error {
	refs := make([]models.SecretRef, 0, len(bindings))
	for alias, ref := range bindings {
		if !models.ValidSecretName(alias) || !ref.Scope.Valid() || !models.ValidSecretName(ref.Name) {
			return ErrInvalidBinding
		}
		refs = append(refs, ref)
	}

	return srv.secrets.CanBind(principal, boardID, refs)
}

func (srv *PostItsService) MovePostIt(principal models.Principal, id uuid.UUID, pos models.Position) error {
	postit, err := srv.load(principal, id, models.BoardEditor)
	if err != nil {
		return err
	}

	return srv.db.MovePostIt(postit.Board, postit.Id, pos)
}

func (srv *PostItsService) DeletePostIt(principal models.Principal, id uuid.UUID) ([]models.Strand, error) {
	if _, err := srv.load(principal, id, models.BoardEditor); err != nil {
		return nil, err
	}

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

// refsOf maps every token the card uses to where it points: an explicit
// binding, or the board's own scope by that name.
func refsOf(postit *models.PostIts) map[string]models.SecretRef {
	refs := make(map[string]models.SecretRef)
	for _, token := range secretRefs(postit) {
		if ref, bound := postit.Bindings[token]; bound {
			refs[token] = ref
			continue
		}
		refs[token] = models.SecretRef{Scope: models.BoardScope(postit.Board), Name: token}
	}
	return refs
}

func (srv *PostItsService) runsAs(postit *models.PostIts) (models.Principal, error) {
	if postit.RunAs != "" {
		return models.Principal{ID: postit.RunAs}, nil
	}

	board, err := srv.db.FindBoard(postit.Board)
	if err != nil {
		return models.Principal{}, err
	}
	return models.Principal{ID: board.Owner}, nil
}

// userParams resolves the card's tokens as whoever it runs as. A board token
// nobody stored stays verbatim; an explicit binding that cannot be honoured
// stops the card.
func (srv *PostItsService) userParams(postit *models.PostIts) (map[string]string, error) {
	refs := refsOf(postit)
	if len(refs) == 0 {
		return nil, nil
	}

	principal, err := srv.runsAs(postit)
	if err != nil {
		return nil, err
	}

	wanted := make([]models.SecretRef, 0, len(refs))
	for _, ref := range refs {
		wanted = append(wanted, ref)
	}

	resolved, err := srv.secrets.ResolveAs(principal, postit.Board, wanted)
	if errors.Is(err, infrastructure.ErrForbidden) {
		return nil, ErrCredentialUnavailable
	}
	if err != nil {
		return nil, err
	}

	values := make(map[string]string, len(resolved))
	for token, ref := range refs {
		value, found := resolved[ref]
		if _, bound := postit.Bindings[token]; bound && !found {
			return nil, ErrCredentialUnavailable
		}
		if found {
			values["$"+token] = value
		}
	}

	return values, nil
}

// systemParams resolves the platform credentials a well-known declares, keyed
// by the placeholder its template uses.
func (srv *PostItsService) systemParams(postit *models.PostIts) (map[string]string, error) {
	def, ok := configuredPostIts[postit.WellKnown]
	if !ok || len(def.systemSecrets) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(def.systemSecrets))
	for _, name := range def.systemSecrets {
		names = append(names, string(name))
	}

	resolved, err := srv.secrets.Resolve(models.SystemScope, names)
	if err != nil {
		return nil, err
	}

	params := make(map[string]string, len(def.systemSecrets))
	for placeholder, name := range def.systemSecrets {
		value, ok := resolved["$"+string(name)]
		if !ok {
			log.Printf("well-known %s: system secret %s is not configured", postit.WellKnown, name)
			return nil, ErrSystemSecretMissing
		}
		params[placeholder] = value
	}

	return params, nil
}

func (srv *PostItsService) prepare(postit *models.PostIts) (*models.PostIts, error) {
	resolved, err := srv.userParams(postit)
	if err != nil {
		return nil, err
	}

	system, err := srv.systemParams(postit)
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
	// Last, so nothing the user can edit shadows a platform credential.
	maps.Copy(clone.Params, system)

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
