package realtime

import (
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	bsrv "github.com/Secreto31126/tesis/common/services/boards"
	"github.com/google/uuid"
)

type RealTimeService struct {
	boards *bsrv.BoardService
	cache  infrastructure.Cache
}

func New(boards *bsrv.BoardService, cache infrastructure.Cache) *RealTimeService {
	return &RealTimeService{boards, cache}
}

func (srv *RealTimeService) AddClientOnline(principal models.Principal, boardID, client uuid.UUID) ([]string, error) {
	board, err := srv.boards.GetBoard(principal, boardID)
	if err != nil {
		return nil, err
	}

	return srv.cache.ConnectClientToBoard(board, client)
}

func (srv *RealTimeService) RemoveClientOnline(principal models.Principal, boardID, client uuid.UUID) error {
	board, err := srv.boards.GetBoard(principal, boardID)
	if err != nil {
		return err
	}

	return srv.cache.DisconnectClientFromBoard(board, client)
}
