package chat

import "time"

type ChatModel struct {
	LeftoverID string    `db:"leftover_id"`
	UserID     string    `db:"user_id"`
	Message    string    `db:"message"`
	Image      string    `db:"image"`
	CreatedAt  time.Time `db:"created_at"`
	IsSeen     bool      `db:"is_seen"`
}
