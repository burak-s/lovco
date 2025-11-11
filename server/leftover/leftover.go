package leftover

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	addLeftoverQuery = `
		INSERT INTO leftover (id, owner_id, name, description, type, image_url, longitude, latitude, street, district, city, province, state, country) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);
	`
	getLeftoverQuery = `
		SELECT id, owner_id, name, description, type, image_url, longitude, latitude, street, district, city, province, state, country
		FROM leftover
		WHERE id = $1;
	`

	searchLeftoversQuery = `
		SELECT id, owner_id, name, description, type, image_url, longitude, latitude, street, district, city, province, state, country
		FROM leftover
	`

	updateLeftoverQuery = `
		UPDATE leftover
		SET owner_id = $1, name = $2, description = $3, image_url = $4, longitude = $5, latitude = $6, street = $7, district = $8, city = $9, province = $10, state = $11, country = $12
		WHERE id = $13;
	`
	deleteLeftoverQuery = `
		DELETE FROM leftover
		WHERE id = $1 AND owner_id = $2;
	`
)

type DatabaseInterface interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func buildLeftoverSelectQuery(req *LeftoverQuery) (string, []any) {
	conds := make([]string, 0)
	args := make([]any, 0)
	argIdx := 1

	if req.OwnerId != nil {
		conds = append(conds, fmt.Sprintf("owner_id = $%d", argIdx))
		args = append(args, req.OwnerId)
		argIdx++
	}
	if req.Name != nil {
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, "%"+*req.Name+"%")
		argIdx++
	}

	if req.Type != nil {
		conds = append(conds, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, req.Type)
		argIdx++
	}

	if req.Bbox != nil {
		conds = append(conds, fmt.Sprintf("longitude >= $%d AND longitude <= $%d AND latitude >= $%d AND latitude <= $%d", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, req.Bbox.TopLeft.Longitude, req.Bbox.BottomRight.Longitude, req.Bbox.TopLeft.Latitude, req.Bbox.BottomRight.Latitude)
		argIdx += 4
	}

	query := searchLeftoversQuery
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	return query, args
}

type LeftoverServer struct {
	UnimplementedLeftoverServiceServer
	db DatabaseInterface
}

func NewLeftoverServer(db *pgxpool.Pool) *LeftoverServer {
	return &LeftoverServer{
		db: db,
	}
}

func (s *LeftoverServer) AddLeftover(ctx context.Context, req *LeftoverRequest) (*emptypb.Empty, error) {
	id := uuid.New()

	_, err := s.db.Exec(ctx, addLeftoverQuery, id, req.OwnerId, req.Name, req.Description, req.Type, req.ImageUrl, req.Coordinates.Longitude, req.Coordinates.Latitude, req.Address.Street, req.Address.District, req.Address.City, req.Address.Province, req.Address.State, req.Address.Country)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *LeftoverServer) GetLeftover(ctx context.Context, req *LeftoverIdentity) (*Leftover, error) {
	row := s.db.QueryRow(ctx, getLeftoverQuery, req.Id)
	var lo Leftover
	err := row.Scan(
		&lo.Id,
		&lo.OwnerId,
		&lo.Name,
		&lo.Description,
		&lo.Type,
		&lo.ImageUrl,
		&lo.Coordiantes.Longitude, &lo.Coordiantes.Latitude,
		&lo.Address.Street, &lo.Address.District, &lo.Address.City, &lo.Address.Province, &lo.Address.State, &lo.Address.Country)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, status.Errorf(codes.NotFound, "leftover not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get leftover: %v", err)
	}

	return &Leftover{
		Id:          lo.Id,
		OwnerId:     lo.OwnerId,
		Name:        lo.Name,
		Description: lo.Description,
		Type:        lo.Type,
		ImageUrl:    lo.ImageUrl,
		Coordiantes: &Point{
			Longitude: lo.Coordiantes.Longitude,
			Latitude:  lo.Coordiantes.Latitude,
		},
		Address: &Address{
			Street:   lo.Address.Street,
			District: lo.Address.District,
			City:     lo.Address.City,
			Province: lo.Address.Province,
			State:    lo.Address.State,
			Country:  lo.Address.Country,
		},
	}, nil
}

func (s *LeftoverServer) GetLeftovers(ctx context.Context, req *LeftoverQuery) (*LeftoverResponse, error) {
	items := make([]*Leftover, 0)
	query, args := buildLeftoverSelectQuery(req)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query leftovers: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var lo Leftover
		err := rows.Scan(&lo.Id, &lo.OwnerId, &lo.Name, &lo.Description, &lo.Type, &lo.ImageUrl, &lo.Coordiantes.Longitude, &lo.Coordiantes.Latitude, &lo.Address.Street, &lo.Address.District, &lo.Address.City, &lo.Address.Province, &lo.Address.State, &lo.Address.Country)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan leftover: %v", err)
		}
		items = append(items, &Leftover{
			Id:          lo.Id,
			OwnerId:     lo.OwnerId,
			Name:        lo.Name,
			Description: lo.Description,
			Type:        lo.Type,
			ImageUrl:    lo.ImageUrl,
			Coordiantes: &Point{
				Longitude: lo.Coordiantes.Longitude,
				Latitude:  lo.Coordiantes.Latitude,
			},
			Address: &Address{
				Street:   lo.Address.Street,
				District: lo.Address.District,
				City:     lo.Address.City,
				Province: lo.Address.Province,
				State:    lo.Address.State,
				Country:  lo.Address.Country,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "error iterating rows: %v", err)
	}

	return &LeftoverResponse{Items: items}, nil
}

func (s *LeftoverServer) UpdateLeftover(ctx context.Context, req *Leftover) (*emptypb.Empty, error) {
	uid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}
	_, err = s.db.Exec(ctx, updateLeftoverQuery,
		req.OwnerId,
		req.Name,
		req.Description,
		req.ImageUrl,
		req.Coordiantes.Longitude, req.Coordiantes.Latitude,
		req.Address.Street, req.Address.District, req.Address.City, req.Address.Province, req.Address.State, req.Address.Country, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *LeftoverServer) DeleteLeftover(ctx context.Context, req *DeleteRequest) (*emptypb.Empty, error) {
	_, err := s.db.Exec(ctx, deleteLeftoverQuery, req.Id, req.OwnerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete leftover: %v", err)
	}

	return &emptypb.Empty{}, nil
}
