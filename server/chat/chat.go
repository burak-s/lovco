package chat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DatabaseInterface interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
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
	ownerID     string
	guestID     string
	queue       []waiter          // queue for users waiting for a slot
	broadcaster chan *ChatMessage // broadcast channel for messages
	closed      bool              // if the room is closed
}

// artificial queue for business logic. Users are waiting for a slot
// uid is the user id
// stream is the stream for the user
// ready is a channel that is closed when the user is ready to be added to the slots
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

func joinRoom(roomID string, uid string, isOwner bool, stream ChatService_JoinChatServer) error {
	slog.Info("user is trying to join room", "user_id", uid, "leftover_id", roomID, "is_owner", isOwner)
	// lock room map to prevent race conditions
	room := getRoom(roomID)

	// lock room to prevent race conditions
	room.mu.Lock()
	if room.closed {
		// if room is closed unlock, return error
		room.mu.Unlock()
		return status.Errorf(codes.Canceled, "chat session is closed")
	}

	// owner can join room a seat is always available for them
	if isOwner {
		slog.Info("user is owner, joining room", "user_id", uid, "leftover_id", roomID)
		room.ownerID = uid
		room.slots[uid] = stream
		room.mu.Unlock()
		return nil
	}

	// if there is a slot available, add to slots and return
	if room.guestID == "" || room.guestID == uid {
		slog.Info("user is guest, joining room", "user_id", uid, "leftover_id", roomID)
		room.guestID = uid
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
	slog.Info("user is guest, joining queue", "user_id", uid, "leftover_id", roomID)
	room.mu.Unlock()

	// Wait for a slot to be available
	<-queuedWaiter.ready

	return nil
}

func leaveRoom(roomID string, uid string) {
	slog.Info("user is leaving room", "user_id", uid, "leftover_id", roomID)
	// lock room map to prevent race conditions
	roomsMu.Lock()
	room := rooms[roomID]
	roomsMu.Unlock()

	// lock room to prevent race conditions
	room.mu.Lock()

	// remove user from slots
	delete(room.slots, uid)

	// if user is guest, remove them from room definition
	if room.guestID == uid {
		slog.Info("user is guest, removing from room definition", "user_id", uid, "leftover_id", roomID)
		room.guestID = ""
	}

	// if there is a queue, remove the first user from the queue and add them to the slots
	if len(room.queue) > 0 {
		slog.Info("another user is joining room", "user_id", uid, "leftover_id", roomID)
		nextWaiter := room.queue[0]
		room.queue = room.queue[1:]
		room.guestID = nextWaiter.uid
		// keep the slot
		room.slots[nextWaiter.uid] = nextWaiter.stream
		close(nextWaiter.ready)
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
		SELECT owner_id 
		FROM leftover
		WHERE id = $1
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

	isOwner, err := isUserOwner(ctx, s.db, uid, lid)
	if err != nil {
		return err
	}

	// try to join room
	err = joinRoom(lid, uid, isOwner, stream)
	if err != nil {
		return err
	}
	defer leaveRoom(lid, uid)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (s *ChatServer) WatchChatQueue(req *JoinChatRequest, stream ChatService_WatchChatQueueServer) error {
	uid := req.UserId
	lid := req.LeftoverId
	ctx := stream.Context()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			var queuedCount int
			var position int32 = -1

			roomsMu.RLock()
			room := rooms[lid]
			roomsMu.RUnlock()

			if room != nil {
				room.mu.Lock()
				if room.closed {
					room.mu.Unlock()
					return status.Errorf(codes.Canceled, "chat session is closed")
				}

				queuedCount = len(room.queue)
				for i, w := range room.queue {
					if w.uid == uid {
						position = int32(i + 1)
						break
					}
				}

				if position == -1 {
					if _, ok := room.slots[uid]; ok {
						position = 0
					}
				}
				room.mu.Unlock()
			} else {
				queuedCount = 0
				position = -1
			}

			if err := stream.Send(&QueueResponse{
				QueuedCount: int32(queuedCount),
				Position:    position,
			}); err != nil {
				return err
			}
		}
	}
}

func (s *ChatServer) SendMessage(ctx context.Context, req *ChatMessageRequest) (*emptypb.Empty, error) {
	roomsMu.RLock()
	room := rooms[req.LeftoverId]
	roomsMu.RUnlock()
	if room != nil {
		room.broadcaster <- &ChatMessage{
			LeftoverId: req.LeftoverId,
			UserId:     req.UserId,
			Message:    req.Message,
			Image:      req.Image,
			CreatedAt:  timestamppb.Now(),
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) EndChatSession(ctx context.Context, req *EndChatRequest) (*emptypb.Empty, error) {
	isOwner, err := isUserOwner(ctx, s.db, req.UserId, req.LeftoverId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get leftover owner: %v", err)
	}

	if !isOwner {
		slog.Info("user is not owner, leaving room", "user_id", req.UserId, "leftover_id", req.LeftoverId)
		leaveRoom(req.LeftoverId, req.UserId)
		return &emptypb.Empty{}, nil
	}

	slog.Info("user is owner, ending chat session", "user_id", req.UserId, "leftover_id", req.LeftoverId)

	return &emptypb.Empty{}, nil
}
