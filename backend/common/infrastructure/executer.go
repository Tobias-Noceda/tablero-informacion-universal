package infrastructure

import (
	"errors"
	"fmt"

	"github.com/Secreto31126/tesis/common/models"
)

type Executer interface {
	Execute(postit *models.PostIts) (any, error)
}

// Why a card could not be run, so the board can say what to check. The
// executer wraps the cause in one of these; the message keeps the detail.
var (
	ErrSourceTimeout     = errors.New("The card's source did not answer in time")
	ErrSourceUnreachable = errors.New("The card's source could not be reached")
	ErrSourceUnreadable  = errors.New("The card's source answered something that could not be read")
	ErrQueryFailed       = errors.New("The card's query failed")
)

// SourceStatusError is a card's source answering anything but 200.
type SourceStatusError struct {
	Status int
}

func (e *SourceStatusError) Error() string {
	return fmt.Sprintf("Resource returned %d", e.Status)
}
