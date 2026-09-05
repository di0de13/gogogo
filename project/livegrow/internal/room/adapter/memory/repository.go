package memory

import (
	"context"
	"sync"

	"livegrow/internal/room/domain"
)

type Repository struct {
	mu    sync.RWMutex
	rooms map[string]domain.Room
}

func NewRepository(rooms ...domain.Room) *Repository {
	r := &Repository{rooms: make(map[string]domain.Room, len(rooms))}
	for _, room := range rooms {
		if room.Validate() == nil {
			r.rooms[room.ID] = room
		}
	}
	return r
}

func (r *Repository) Get(ctx context.Context, roomID string) (domain.Room, error) {
	if err := ctx.Err(); err != nil {
		return domain.Room{}, err
	}
	r.mu.RLock()
	room, ok := r.rooms[roomID]
	r.mu.RUnlock()
	if !ok {
		return domain.Room{}, domain.ErrRoomNotFound
	}
	return room, nil
}

func (r *Repository) Put(room domain.Room) error {
	if err := room.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	r.rooms[room.ID] = room
	r.mu.Unlock()
	return nil
}
