package application

import (
	"context"
	"testing"

	"livegrow/internal/room/domain"
)

type fakeRoomRepository struct {
	room domain.Room
	err  error
}

func (f fakeRoomRepository) Get(context.Context, string) (domain.Room, error) {
	return f.room, f.err
}

func TestGetRoomUsesInjectedRepository(t *testing.T) {
	want := domain.Room{ID: "room-1", Region: "sg", Live: true}
	service := NewService(fakeRoomRepository{room: want})
	got, err := service.GetRoom(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if got != want {
		t.Fatalf("GetRoom() = %+v, want %+v", got, want)
	}
}

func TestGetRoomPropagatesRepositoryError(t *testing.T) {
	wantErr := domain.ErrRoomNotFound
	service := NewService(fakeRoomRepository{err: wantErr})
	if _, err := service.GetRoom(context.Background(), "missing"); err != wantErr {
		t.Fatalf("GetRoom() error = %v, want %v", err, wantErr)
	}
}
