package domain

import (
	"errors"
	"strings"
)

var (
	ErrRoomIDRequired = errors.New("room id is required")
	ErrRoomNotFound   = errors.New("room not found")
)

type Room struct {
	ID          string
	Title       string
	Region      string
	Live        bool
	ViewerCount int64
}

func (r Room) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return ErrRoomIDRequired
	}
	if strings.TrimSpace(r.Region) == "" {
		return errors.New("room region is required")
	}
	if r.ViewerCount < 0 {
		return errors.New("viewer count cannot be negative")
	}
	return nil
}

func (r Room) IsAvailable() bool { return r.Live }
