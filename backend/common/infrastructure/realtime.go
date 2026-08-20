package infrastructure

import (
	"context"

	"github.com/Secreto31126/tesis/common/models"
)

type Realtime interface {
	// Watch for boards updates
	WatchBoards(ctx context.Context) (models.Stream[models.Board], error)
}
