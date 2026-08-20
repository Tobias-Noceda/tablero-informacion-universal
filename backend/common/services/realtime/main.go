package realtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type Hub struct {
	mu    sync.RWMutex
	peers map[models.ClientMessaging]bool
}

type RealtimeService struct {
	mu   sync.RWMutex
	rt   infrastructure.Realtime
	hubs map[uuid.UUID]*Hub
	ctx  context.Context
	db   *struct {
		stream models.Stream[models.Board]
		cancel context.CancelFunc
	}
}

func New(rt infrastructure.Realtime, ctx context.Context) *RealtimeService {
	this := &RealtimeService{}
	this.rt = rt
	this.ctx = ctx

	return this
}

func (srv *RealtimeService) AddClient(id uuid.UUID, peer models.ClientMessaging) error {
	hub := srv.getOrCreateHub(id)

	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.peers[peer] = true

	return srv.assertStream()
}

func (srv *RealtimeService) RemoveClient(id uuid.UUID, peer models.ClientMessaging) {
	hub, ok := srv.hubs[id]
	if !ok {
		return
	}

	hub.mu.Lock()
	defer hub.mu.Unlock()

	delete(hub.peers, peer)

	if len(hub.peers) == 0 {
		delete(srv.hubs, id)
		srv.relaxStream()
	}
}

func (srv *RealtimeService) broadcast(board *models.Board) error {
	hub, ok := srv.hubs[board.Id]
	if !ok {
		return fmt.Errorf("Missing hub to broadcast")
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for peer := range hub.peers {
		if err := peer.WriteJSON(board); err != nil {
			// srv.RemoveClient(board.Id, peer) // This is a dead-lock
			continue
		}
	}

	return nil
}

func (srv *RealtimeService) getOrCreateHub(id uuid.UUID) *Hub {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	hub, ok := srv.hubs[id]
	if !ok {
		hub = &Hub{}
		srv.hubs[id] = hub
	}

	return hub
}

func (srv *RealtimeService) assertStream() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	if srv.db != nil {
		return nil
	}

	ctx, cancel := context.WithCancel(srv.ctx)
	stream, err := srv.rt.WatchBoards(ctx)
	if err != nil {
		cancel()
		return err
	}

	srv.db.stream = stream
	srv.db.cancel = cancel

	go func() {
		for stream.Next(ctx) {
			board, err := stream.Get()
			if err != nil {
				continue
			}

			srv.broadcast(board)
		}
	}()

	return nil
}

func (srv *RealtimeService) relaxStream() {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	if srv.db == nil || len(srv.hubs) != 0 {
		return
	}

	srv.db.cancel()
	srv.db = nil
}
