package mocks

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// Default error
var ErrCacheMiss = errors.New("cache miss")

// MockDB is a configurable test double for infrastructure.Database.
type MockDB struct {
	FindUserBoardsFn              func(cognitoID string) ([]models.Board, error)
	FindBoardPostItsFn            func(id uuid.UUID) ([]models.PostIts, error)
	FindPostItFn                  func(id uuid.UUID) (*models.PostIts, error)
	FindBoardFn                   func(id uuid.UUID) (*models.Board, error)
	DeletePostItFn                func(id uuid.UUID) error
	DeleteBoardFn                 func(id uuid.UUID) error
	UpdatePostItFn                func(id uuid.UUID, set map[string]any) error
	CreatePostItFn                func(postIt *models.PostIts, ptype string, pos models.Position) (*models.PostIts, error)
	CreateBoardFn                 func(name, owner string) (*models.Board, error)
	AddCollaboratorToBoardFn      func(boardID uuid.UUID, cognitoID string) error
	RemoveCollaboratorFromBoardFn func(boardID uuid.UUID, cognitoID string) error
	DisconnectPostItsFn           func(boardID, source, target uuid.UUID) error
	ConnectPostItsFn              func(boardID, source, target uuid.UUID) error
	MovePostItFn                  func(boardID, postItID uuid.UUID, pos models.Position) error
	UpdateBoardNameFn             func(id uuid.UUID, name string) error
}

var _ infrastructure.Database = (*MockDB)(nil)

func (m *MockDB) FindUserBoards(cognitoID string) ([]models.Board, error) {
	if m.FindUserBoardsFn != nil {
		return m.FindUserBoardsFn(cognitoID)
	}
	return nil, nil
}

func (m *MockDB) FindBoardPostIts(id uuid.UUID) ([]models.PostIts, error) {
	if m.FindBoardPostItsFn != nil {
		return m.FindBoardPostItsFn(id)
	}
	return nil, nil
}

func (m *MockDB) FindPostIt(id uuid.UUID) (*models.PostIts, error) {
	if m.FindPostItFn != nil {
		return m.FindPostItFn(id)
	}
	return nil, nil
}

func (m *MockDB) FindBoard(id uuid.UUID) (*models.Board, error) {
	if m.FindBoardFn != nil {
		return m.FindBoardFn(id)
	}
	return nil, nil
}

func (m *MockDB) DeletePostIt(id uuid.UUID) error {
	if m.DeletePostItFn != nil {
		return m.DeletePostItFn(id)
	}
	return nil
}

func (m *MockDB) DeleteBoard(id uuid.UUID) error {
	if m.DeleteBoardFn != nil {
		return m.DeleteBoardFn(id)
	}
	return nil
}

func (m *MockDB) UpdatePostIt(id uuid.UUID, set map[string]any) error {
	if m.UpdatePostItFn != nil {
		return m.UpdatePostItFn(id, set)
	}
	return nil
}

func (m *MockDB) CreatePostIt(postIt *models.PostIts, ptype string, pos models.Position) (*models.PostIts, error) {
	if m.CreatePostItFn != nil {
		return m.CreatePostItFn(postIt, ptype, pos)
	}
	return postIt, nil
}

func (m *MockDB) CreateBoard(name, owner string) (*models.Board, error) {
	if m.CreateBoardFn != nil {
		return m.CreateBoardFn(name, owner)
	}
	return nil, nil
}

func (m *MockDB) AddCollaboratorToBoard(boardID uuid.UUID, cognitoID string) error {
	if m.AddCollaboratorToBoardFn != nil {
		return m.AddCollaboratorToBoardFn(boardID, cognitoID)
	}
	return nil
}

func (m *MockDB) RemoveCollaboratorFromBoard(boardID uuid.UUID, cognitoID string) error {
	if m.RemoveCollaboratorFromBoardFn != nil {
		return m.RemoveCollaboratorFromBoardFn(boardID, cognitoID)
	}
	return nil
}

func (m *MockDB) DisconnectPostIts(boardID, source, target uuid.UUID) error {
	if m.DisconnectPostItsFn != nil {
		return m.DisconnectPostItsFn(boardID, source, target)
	}
	return nil
}

func (m *MockDB) ConnectPostIts(boardID, source, target uuid.UUID) error {
	if m.ConnectPostItsFn != nil {
		return m.ConnectPostItsFn(boardID, source, target)
	}
	return nil
}

func (m *MockDB) MovePostIt(boardID, postItID uuid.UUID, pos models.Position) error {
	if m.MovePostItFn != nil {
		return m.MovePostItFn(boardID, postItID, pos)
	}
	return nil
}

func (m *MockDB) UpdateBoardName(id uuid.UUID, name string) error {
	if m.UpdateBoardNameFn != nil {
		return m.UpdateBoardNameFn(id, name)
	}
	return nil
}

// MockCache is a configurable test double for infrastructure.Cache.
type MockCache struct {
	FindPostItResultFn          func(id uuid.UUID) (any, error)
	AddPostItResultFn           func(postit *models.PostIts, data any) error
	DropPostItResultFn          func(id uuid.UUID) error
	ConnectClientToBoardFn      func(board *models.Board, id uuid.UUID) ([]string, error)
	DisconnectClientFromBoardFn func(board *models.Board, id uuid.UUID) error
}

var _ infrastructure.Cache = (*MockCache)(nil)

func (m *MockCache) FindPostItResult(id uuid.UUID) (any, error) {
	if m.FindPostItResultFn != nil {
		return m.FindPostItResultFn(id)
	}
	return nil, ErrCacheMiss
}

func (m *MockCache) AddPostItResult(postit *models.PostIts, data any) error {
	if m.AddPostItResultFn != nil {
		return m.AddPostItResultFn(postit, data)
	}
	return nil
}

func (m *MockCache) DropPostItResult(id uuid.UUID) error {
	if m.DropPostItResultFn != nil {
		return m.DropPostItResultFn(id)
	}
	return nil
}

func (m *MockCache) ConnectClientToBoard(board *models.Board, id uuid.UUID) ([]string, error) {
	if m.ConnectClientToBoardFn != nil {
		return m.ConnectClientToBoardFn(board, id)
	}
	return nil, ErrCacheMiss
}

func (m *MockCache) DisconnectClientFromBoard(board *models.Board, id uuid.UUID) error {
	if m.DisconnectClientFromBoardFn != nil {
		return m.DisconnectClientFromBoardFn(board, id)
	}
	return nil
}

// MockExecuter is a configurable test double for infrastructure.Executer.
type MockExecuter struct {
	ExecuteFn func(postit *models.PostIts) (any, error)
}

var _ infrastructure.Executer = (*MockExecuter)(nil)

func (m *MockExecuter) Execute(postit *models.PostIts) (any, error) {
	if m.ExecuteFn != nil {
		return m.ExecuteFn(postit)
	}
	return nil, nil
}

type MockSecretResolver struct {
	ResolveFn func(board uuid.UUID, names []string) (map[string]string, error)
}

var _ infrastructure.SecretResolver = (*MockSecretResolver)(nil)

func (m *MockSecretResolver) Resolve(board uuid.UUID, names []string) (map[string]string, error) {
	if m.ResolveFn != nil {
		return m.ResolveFn(board, names)
	}
	return nil, nil
}

type MockSecretStore struct {
	UpsertSecretFn func(secret *models.Secret) error
	FindSecretsFn  func(board uuid.UUID, names []string) ([]models.Secret, error)
	ListSecretsFn  func(board uuid.UUID) ([]models.Secret, error)
	DeleteSecretFn func(board uuid.UUID, name string) error
}

var _ infrastructure.SecretStore = (*MockSecretStore)(nil)

func (m *MockSecretStore) UpsertSecret(secret *models.Secret) error {
	if m.UpsertSecretFn != nil {
		return m.UpsertSecretFn(secret)
	}
	return nil
}

func (m *MockSecretStore) FindSecrets(board uuid.UUID, names []string) ([]models.Secret, error) {
	if m.FindSecretsFn != nil {
		return m.FindSecretsFn(board, names)
	}
	return nil, nil
}

func (m *MockSecretStore) ListSecrets(board uuid.UUID) ([]models.Secret, error) {
	if m.ListSecretsFn != nil {
		return m.ListSecretsFn(board)
	}
	return nil, nil
}

func (m *MockSecretStore) DeleteSecret(board uuid.UUID, name string) error {
	if m.DeleteSecretFn != nil {
		return m.DeleteSecretFn(board, name)
	}
	return nil
}

// MockTokenClient is a configurable test double for infrastructure.TokenClient.
type MockTokenClient struct {
	FetchFn    func(material *models.OAuth2Material) error
	ExchangeFn func(material *models.OAuth2Material, code, redirectURI, verifier string) error
}

var _ infrastructure.TokenClient = (*MockTokenClient)(nil)

func (m *MockTokenClient) Fetch(material *models.OAuth2Material) error {
	if m.FetchFn != nil {
		return m.FetchFn(material)
	}
	return nil
}

func (m *MockTokenClient) Exchange(material *models.OAuth2Material, code, redirectURI, verifier string) error {
	if m.ExchangeFn != nil {
		return m.ExchangeFn(material, code, redirectURI, verifier)
	}
	return nil
}

// MockLocker is a configurable test double for infrastructure.Locker. It grants
// the lock unless told otherwise.
type MockLocker struct {
	AcquireFn func(key string, ttl time.Duration) (string, bool, error)
	ReleaseFn func(key, token string) error
}

var _ infrastructure.Locker = (*MockLocker)(nil)

func (m *MockLocker) Acquire(key string, ttl time.Duration) (string, bool, error) {
	if m.AcquireFn != nil {
		return m.AcquireFn(key, ttl)
	}
	return "token", true, nil
}

func (m *MockLocker) Release(key, token string) error {
	if m.ReleaseFn != nil {
		return m.ReleaseFn(key, token)
	}
	return nil
}

// MockHandshakeStore is an in-memory infrastructure.HandshakeStore. Take
// removes the entry, mirroring the single-use guarantee.
type MockHandshakeStore struct {
	entries map[string][]byte
}

var _ infrastructure.HandshakeStore = (*MockHandshakeStore)(nil)

func (m *MockHandshakeStore) Put(key string, value []byte, ttl time.Duration) error {
	if m.entries == nil {
		m.entries = make(map[string][]byte)
	}
	m.entries[key] = value
	return nil
}

func (m *MockHandshakeStore) Take(key string) ([]byte, error) {
	value, ok := m.entries[key]
	if !ok {
		return nil, errors.New("no such handshake")
	}
	delete(m.entries, key)
	return value, nil
}
