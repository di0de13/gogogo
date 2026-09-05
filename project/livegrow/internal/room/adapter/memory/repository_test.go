package memory

import (
	"context"
	"testing"

	"livegrow/internal/room/domain"
)

func TestRepositoryGetAndPut(t *testing.T) {
	repository := NewRepository()
	room := domain.Room{ID: "room-1", Region: "us", Live: true}
	if err := repository.Put(room); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	got, err := repository.Get(context.Background(), room.ID)
	if err != nil || got != room {
		t.Fatalf("Get() = %+v, %v; want %+v", got, err, room)
	}
}

func TestRepositoryHonorsCanceledContext(t *testing.T) {
	repository := NewRepository(domain.Room{ID: "room-1", Region: "us"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repository.Get(ctx, "room-1"); err == nil {
		t.Fatal("Get() expected context error")
	}
}
