package chat

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DatabaseInterface interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type chatModel struct {
	LeftoverID string    `db:"leftover_id"`
	UserID     string    `db:"user_id"`
	Message    string    `db:"message"`
	Image      string    `db:"image"`
	IsSeen     bool      `db:"is_seen"`
	CreatedAt  time.Time `db:"created_at"`
}

// A room is a limited private chat between a user and a leftover owner.
// Cannot be seen by others until they are entered to room.
// Once the user enters the room, they can see all the chat history.
// Once the user leaves the room, they can no longer see the chat history.
// Once the leftover owner leaves the room, the room is deleted.
// Once the user leaves the room, the room stay still.
type room struct {
	mu          sync.Mutex // lock for slots and queue
	slots       map[string]ChatService_JoinChatServer
	queue       []waiter          // queue for users waiting for a slot
	broadcaster chan *ChatMessage // broadcast channel for messages
	closed      bool              // if the room is closed
}

type waiter struct {
	uid    string
	stream ChatService_JoinChatServer
	ready  chan struct{}
}

var (
	rooms   = make(map[string]*room)
	roomsMu sync.RWMutex
)

func getRoom(roomID string) *room {
	roomsMu.RLock()
	r := rooms[roomID]
	roomsMu.RUnlock()

	// room is not created, create it
	roomsMu.Lock()
	if r == nil {
		r = &room{
			slots:       make(map[string]ChatService_JoinChatServer),
			broadcaster: make(chan *ChatMessage),
		}
		rooms[roomID] = r
		go r.runBroadcaster()
	}
	roomsMu.Unlock()

	return r
}

func joinRoom(roomID string, uid string, stream ChatService_JoinChatServer) error {
	// lock room map to prevent race conditions
	room := getRoom(roomID)

	// lock room to prevent race conditions
	room.mu.Lock()
	if room.closed {
		// if room is closed unlock, return error
		room.mu.Unlock()
		return status.Errorf(codes.Canceled, "chat session is closed")
	}

	// if there is a slot available, add to slots and return
	// need to add logic for room owner.
	if len(room.slots) < 2 {
		room.slots[uid] = stream
		room.mu.Unlock()
		return nil
	}

	// Not enough slots, add to queue
	queuedWaiter := waiter{
		uid:    uid,
		stream: stream,
		ready:  make(chan struct{}),
	}
	room.queue = append(room.queue, queuedWaiter)
	room.mu.Unlock()

	// Wait for a slot to be available
	<-queuedWaiter.ready

	return nil
}

func leaveRoom(db DatabaseInterface, roomID string, uid string) {
	roomsMu.Lock()
	room := rooms[roomID]
	roomsMu.Unlock()

	isOwner, err := isUserOwner(context.Background(), db, uid, roomID)
	if err != nil {
		return
	}

	room.mu.Lock()
	delete(room.slots, uid)

	if isOwner {
		roomsMu.Lock()
		room.closed = true
		close(room.broadcaster)
		delete(rooms, roomID)
		roomsMu.Unlock()
	} else {
		if len(room.queue) > 0 {
			nextWaiter := room.queue[0]
			room.queue = room.queue[1:]
			room.slots[nextWaiter.uid] = nextWaiter.stream
			close(nextWaiter.ready)
		}
	}

	room.mu.Unlock()
}

func (room *room) runBroadcaster() {
	for msg := range room.broadcaster {
		room.mu.Lock()
		for uid, stream := range room.slots {
			if err := stream.Send(msg); err != nil {
				delete(room.slots, uid)
			}
		}
		room.mu.Unlock()
	}
}

func isUserOwner(ctx context.Context, db DatabaseInterface, userID string, leftoverID string) (bool, error) {
	query := `
		SELECT owner_id FROM leftover WHERE id = $1
	`
	row := db.QueryRow(ctx, query, leftoverID)
	var ownerID string
	err := row.Scan(&ownerID)
	if err != nil {
		return false, status.Errorf(codes.Internal, "failed to get leftover owner: %v", err)
	}
	return ownerID == userID, nil
}

type ChatServer struct {
	UnimplementedChatServiceServer
	db DatabaseInterface
}

func NewChatServer(db *pgxpool.Pool) *ChatServer {
	return &ChatServer{
		db: db,
	}
}

func (s *ChatServer) JoinChat(req *JoinChatRequest, stream ChatService_JoinChatServer) error {
	uid := req.UserId
	lid := req.LeftoverId
	ctx := stream.Context()

	// try to join room
	err := joinRoom(lid, uid, stream)
	if err != nil {
		return err
	}
	defer leaveRoom(s.db, lid, uid)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var in ChatMessage
			if err := stream.RecvMsg(&in); err != nil {
				return err
			}

			roomsMu.RLock()
			room := rooms[lid]
			roomsMu.RUnlock()

			if room != nil {
				room.broadcaster <- &in
			}
		}
	}
}

func (s *ChatServer) SendMessage(ctx context.Context, req *ChatMessageRequest) (*emptypb.Empty, error) {
	query := `
		INSERT INTO chat_message (leftover_id, user_id, message, image, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := s.db.Exec(ctx, query, req.LeftoverId, req.UserId, req.Message, req.Image, time.Now())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send message: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) EndChatSession(ctx context.Context, req *EndChatRequest) (*emptypb.Empty, error) {
	isOwner, err := isUserOwner(ctx, s.db, req.UserId, req.LeftoverId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get leftover owner: %v", err)
	}

	if !isOwner {
		// leave room
		// 0

		return &emptypb.Empty{}, nil
	}

	query := `
		DELETE FROM chat_message 
		WHERE leftover_id = $1
	`
	_, err = s.db.Exec(ctx, query, req.LeftoverId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to end chat session: %v", err)
	}

	return &emptypb.Empty{}, nil
}
