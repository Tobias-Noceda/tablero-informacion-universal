package infrastructure

import (
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type Database interface {
	// Find user's boards
	FindUserBoards(cognitoId string) ([]models.Board, error)
	// Find all the board post-its
	FindBoardPostIts(id uuid.UUID) ([]models.PostIts, error)
	// Find a given postIt
	FindPostIt(id uuid.UUID) (*models.PostIts, error)
	// Find a given board
	FindBoard(id uuid.UUID) (*models.Board, error)
	// Delete a given PostIt
	DeletePostIt(id uuid.UUID) ([]models.Strand, error)
	// Delete a given Board
	DeleteBoard(id uuid.UUID) error
	// Updates a given PostIt
	UpdatePostIt(id uuid.UUID, set map[string]any) error
	// Creates a PostIt
	CreatePostIt(postIt *models.PostIts, ptype string, pos models.Position) (*models.PostIts, error)
	// Creates a Board
	CreateBoard(name, owner string) (*models.Board, error)
	// Adds collaborator to board
	AddCollaboratorToBoard(boardID uuid.UUID, cognitoID string) error
	// Removes collaborator from board
	RemoveCollaboratorFromBoard(boardID uuid.UUID, cognitoID string) error
	// Disconnects a strand
	DisconnectPostIts(boardID, strandID uuid.UUID) error
	// Connects a Strand
	ConnectPostIts(boardID, source, target uuid.UUID) error
	// Move Post It
	MovePostIt(boardID, postItID uuid.UUID, pos models.Position) error
	// Update board's name
	UpdateBoardName(id uuid.UUID, name string) error
}
