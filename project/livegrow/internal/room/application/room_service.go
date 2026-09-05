package application

import (
	"context"
	"errors"

	"livegrow/internal/room/domain"
)

type RoomRepository interface {
	Get(ctx context.Context, roomID string) (domain.Room, error)
}

type Service struct {
	repository RoomRepository
}

func NewService(repository RoomRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetRoom(ctx context.Context, roomID string) (domain.Room, error) {
	if s == nil || s.repository == nil {
		return domain.Room{}, errors.New("room repository is not configured")
	}
	return s.repository.Get(ctx, roomID)
}
