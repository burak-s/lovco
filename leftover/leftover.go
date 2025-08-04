package leftover

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DatabaseInterface interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type LeftoverModel struct {
	ID          uuid.UUID `db:"id"`
	OwnerID     uuid.UUID `db:"owner_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Image       string    `db:"image"`
	Longitude   float64   `db:"longitude"`
	Latitude    float64   `db:"latitude"`
}

// Inject database into server
type LeftoverServer struct {
	UnimplementedLeftoverServiceServer
	db DatabaseInterface
}

// NewLeftoverServer now accepts database connection
func NewLeftoverServer(db *pgxpool.Pool) *LeftoverServer {
	return &LeftoverServer{
		db: db,
	}
}

func (s *LeftoverServer) AddLeftover(ctx context.Context, req *LeftoverRequest) (*emptypb.Empty, error) {
	id := uuid.New()

	query := `
		INSERT INTO leftover (id, owner_id, name, description, image, longitude, latitude) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.Exec(ctx, query, id, req.OwnerId, req.Name, req.Description, req.Image, req.Longitude, req.Latitude)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *LeftoverServer) GetLeftover(ctx context.Context, req *LeftoverIdentity) (*Leftover, error) {
	query := `
		SELECT id, owner_id, name, description, image, longitude, latitude 
		FROM leftover 
		WHERE id = $1
	`
	row := s.db.QueryRow(ctx, query, req.Id)
	var lo LeftoverModel
	err := row.Scan(&lo.ID, &lo.OwnerID, &lo.Name, &lo.Description, &lo.Image, &lo.Longitude, &lo.Latitude)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, status.Errorf(codes.NotFound, "leftover not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get leftover: %v", err)
	}

	return &Leftover{
		Id:          lo.ID.String(),
		OwnerId:     lo.OwnerID.String(),
		Name:        lo.Name,
		Description: lo.Description,
		Image:       lo.Image,
		Longitude:   float32(lo.Longitude),
		Latitude:    float32(lo.Latitude),
	}, nil
}

func (s *LeftoverServer) GetLeftovers(ctx context.Context, req *LeftoverRequest) (*LeftoverResponse, error) {
	items := make([]*Leftover, 0)

	query := `
		SELECT id, owner_id, name, description, image, longitude, latitude 
		FROM leftover
		WHERE name = $1
	`
	rows, err := s.db.Query(ctx, query, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query leftovers: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lo LeftoverModel
		err := rows.Scan(&lo.ID, &lo.OwnerID, &lo.Name, &lo.Description, &lo.Image, &lo.Longitude, &lo.Latitude)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan leftover: %v", err)
		}
		items = append(items, &Leftover{
			Id:          lo.ID.String(),
			OwnerId:     lo.OwnerID.String(),
			Name:        lo.Name,
			Description: lo.Description,
			Image:       lo.Image,
			Longitude:   float32(lo.Longitude),
			Latitude:    float32(lo.Latitude),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "error iterating rows: %v", err)
	}

	return &LeftoverResponse{Items: items}, nil
}

func (s *LeftoverServer) UpdateLeftover(ctx context.Context, req *Leftover) (*emptypb.Empty, error) {
	query := `UPDATE leftover 
		SET owner_id = $1, name = $2, description = $3, image = $4, longitude = $5, latitude = $6 
		WHERE id = $7
	`

	uid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}
	_, err = s.db.Exec(ctx, query, req.OwnerId, req.Name, req.Description, req.Image, req.Longitude, req.Latitude, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *LeftoverServer) DeleteLeftover(ctx context.Context, req *DeleteRequest) (*emptypb.Empty, error) {
	query := `
		DELETE FROM leftover 
		WHERE id = $1 AND owner_id = $2
	`
	_, err := s.db.Exec(ctx, query, req.Id, req.OwnerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}
