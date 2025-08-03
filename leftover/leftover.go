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
		INSERT INTO leftovers (id, name, description, image, longitude, latitude) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.Exec(ctx, query, id, req.Name, req.Description, req.Image, req.Longitude, req.Latitude)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *LeftoverServer) GetLeftover(ctx context.Context, req *LeftoverIdentity) (*Leftover, error) {
	query := `
		SELECT id, name, description, image, longitude, latitude 
		FROM leftovers 
		WHERE id = $1
	`
	row := s.db.QueryRow(ctx, query, req.Id)
	var lo LeftoverModel
	err := row.Scan(&lo.ID, &lo.Name, &lo.Description, &lo.Image, &lo.Longitude, &lo.Latitude)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, status.Errorf(codes.NotFound, "leftover not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get leftover: %v", err)
	}

	return &Leftover{
		Id:          lo.ID.String(),
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
		SELECT id, name, description, image, longitude, latitude 
		FROM leftovers
		WHERE name = $1
	`
	rows, err := s.db.Query(ctx, query, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query leftovers: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lo LeftoverModel
		err := rows.Scan(&lo.ID, &lo.Name, &lo.Description, &lo.Image, &lo.Longitude, &lo.Latitude)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan leftover: %v", err)
		}
		items = append(items, &Leftover{
			Id:          lo.ID.String(),
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
	query := `UPDATE leftovers 
		SET name = $1, description = $2, image = $3, longitude = $4, latitude = $5 
		WHERE id = $6
	`

	uid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}
	_, err = s.db.Exec(ctx, query, req.Name, req.Description, req.Image, req.Longitude, req.Latitude, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *LeftoverServer) DeleteLeftover(ctx context.Context, req *LeftoverIdentity) (*emptypb.Empty, error) {
	query := `
		DELETE FROM leftovers 
		WHERE id = $1
	`
	_, err := s.db.Exec(ctx, query, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}
