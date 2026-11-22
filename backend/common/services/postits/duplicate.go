package postits

import (
	"log"
	"maps"
	"slices"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// duplicateOffset is how far from the card its copy lands, in canvas pixels:
// enough to see both, close enough to read as the same card.
const duplicateOffset = 40

// DuplicatePostIt copies a card onto its board, next to it: its settings, its
// title and how it looks. It is created like a new card: by an editor, within
// the board's plan, from a card type still offered to them, and it runs as
// whoever duplicated it, so they must be allowed to use every credential it
// is bound to.
func (srv *PostItsService) DuplicatePostIt(principal models.Principal, id uuid.UUID) (*models.BoardPostIt, error) {
	source, err := srv.load(principal, id, models.BoardEditor)
	if err != nil {
		return nil, err
	}

	board, release, err := srv.roomForCard(source.Board)
	if err != nil {
		return nil, err
	}
	defer release()

	i := slices.IndexFunc(board.PostIts, func(p models.BoardPostIt) bool { return p.Id == id })
	if i < 0 {
		return nil, ErrNotAMember
	}
	entry := board.PostIts[i]

	if source.WellKnown != "" {
		if _, err := srv.definition(principal, source.Board, source.WellKnown); err != nil {
			return nil, err
		}
	}
	if err := srv.bind(principal, source.Board, source.Bindings); err != nil {
		return nil, err
	}

	card := copyOf(source)
	card.RunAs = principal.ID
	position := models.Position{X: entry.Position.X + duplicateOffset, Y: entry.Position.Y + duplicateOffset}
	created, err := srv.db.CreatePostIt(card, entry.Title, entry.Type, position)
	if err != nil {
		return nil, err
	}

	copied := models.BoardPostIt{Id: created.Id, Type: entry.Type, Title: entry.Title, Position: position}
	if patch := entry.CardStyle.Patch(); !patch.Empty() {
		// The card exists already: a copy without its look beats a failure.
		if style, err := srv.db.StylePostIt(source.Board, created.Id, patch); err != nil {
			log.Printf("post-it %s: copying the style of %s failed: %v", created.Id, id, err)
		} else {
			copied.CardStyle = *style
		}
	}
	return &copied, nil
}

// copyOf is the card's settings with maps of their own and no id.
func copyOf(source *models.PostIts) *models.PostIts {
	card := *source
	card.Id = uuid.Nil
	card.Params = maps.Clone(source.Params)
	card.Query = maps.Clone(source.Query)
	card.Bindings = maps.Clone(source.Bindings)
	card.Envs = slices.Clone(source.Envs)
	card.Request.Headers = maps.Clone(source.Request.Headers)
	card.Request.Queries = maps.Clone(source.Request.Queries)
	if source.Resource != nil {
		resource := *source.Resource
		card.Resource = &resource
	}
	return &card
}
