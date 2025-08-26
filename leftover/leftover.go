package leftover

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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
	Type        string    `db:"type"` // leftover type e.g. food, electronic,
	Image       string    `db:"image"`
	ImageFormat string    `db:"image_format"`
	Longitude   float64   `db:"longitude"`
	Latitude    float64   `db:"latitude"`
}

func buildLeftoverSelectQuery(req *LeftoverRequest) (string, []interface{}) {
	baseQuery := `
		SELECT id, owner_id, name, description, type, image, image_format, longitude, latitude 
		FROM leftover
	`

	slog.Info("req", "req", req)

	conds := make([]string, 0)
	args := make([]interface{}, 0)
	argIdx := 1

	if req.OwnerId != "" {
		conds = append(conds, fmt.Sprintf("owner_id = $%d", argIdx))
		args = append(args, req.OwnerId)
		argIdx++
	}
	if req.Name != "" {
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, "%"+req.Name+"%")
		argIdx++
	}
	if req.Description != "" {
		conds = append(conds, fmt.Sprintf("description ILIKE $%d", argIdx))
		args = append(args, "%"+req.Description+"%")
		argIdx++
	}
	if req.Image != "" {
		conds = append(conds, fmt.Sprintf("image ILIKE $%d", argIdx))
		args = append(args, "%"+req.Image+"%")
		argIdx++
	}

	if req.Longitude != 0 {
		conds = append(conds, fmt.Sprintf("longitude = $%d", argIdx))
		args = append(args, req.Longitude)
		argIdx++
	}

	if req.Latitude != 0 {
		conds = append(conds, fmt.Sprintf("latitude = $%d", argIdx))
		args = append(args, req.Latitude)
		argIdx++
	}

	query := baseQuery
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	return query, args
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
		INSERT INTO leftover (id, owner_id, name, description, type, image, image_format,  longitude, latitude) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.Exec(ctx, query, id, req.OwnerId, req.Name, req.Description, req.Type, req.Image, req.ImageFormat, req.Longitude, req.Latitude)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *LeftoverServer) GetLeftover(ctx context.Context, req *LeftoverIdentity) (*Leftover, error) {
	query := `
		SELECT id, owner_id, name, description, type, image, image_format, longitude, latitude 
		FROM leftover 
		WHERE id = $1
	`
	row := s.db.QueryRow(ctx, query, req.Id)
	var lo LeftoverModel
	err := row.Scan(&lo.ID, &lo.OwnerID, &lo.Name, &lo.Description, &lo.Type, &lo.Image, &lo.ImageFormat, &lo.Longitude, &lo.Latitude)
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

	query, args := buildLeftoverSelectQuery(req)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query leftovers: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lo LeftoverModel
		err := rows.Scan(&lo.ID, &lo.OwnerID, &lo.Name, &lo.Description, &lo.Type, &lo.Image, &lo.ImageFormat, &lo.Longitude, &lo.Latitude)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan leftover: %v", err)
		}
		items = append(items, &Leftover{
			Id:          lo.ID.String(),
			OwnerId:     lo.OwnerID.String(),
			Name:        lo.Name,
			Description: lo.Description,
			Type:        lo.Type,
			Image:       lo.Image,
			ImageFormat: lo.ImageFormat,
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
