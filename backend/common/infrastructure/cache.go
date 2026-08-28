package infrastructure

import (
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type Cache interface {
	FindPostItResult(id uuid.UUID) (any, error)
	AddPostItResult(postit *models.PostIts, data any) error
	DropPostItResult(id uuid.UUID) error
	ConnectClientToBoard(board *models.Board, id uuid.UUID) ([]string, error)
	DisconnectClientFromBoard(board *models.Board, id uuid.UUID) error
}
