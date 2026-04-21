package event

import "time"

// UserCreatedEvent is published after a user is created.
type UserCreatedEvent struct {
	UserID    int64
	UserName  string
	Email     string
	CreatedAt time.Time
}

// Name returns the stable event name.
func (UserCreatedEvent) Name() string { return "user.created" }
