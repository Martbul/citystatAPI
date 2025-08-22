// internal/repository/visitor.go
package repository

import (
	"context"
	"fmt"

	"citystatAPI/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type VisitorRepository interface {
	GetVisitedStreets(ctx context.Context, userID string) ([]db.VisitedStreet, error)
	CreateVisitedStreet(ctx context.Context, params CreateVisitedStreetParams) (*db.VisitedStreet, error)
	CheckVisitedStreetExists(ctx context.Context, userID, sessionID, streetID string, entryTimestamp int64) (bool, error)
	GetVisitedStreetsBySession(ctx context.Context, userID, sessionID string) ([]db.VisitedStreet, error)
}

type CreateVisitedStreetParams struct {
	UserID          string
	SessionID       string
	StreetID        string
	StreetName      string
	EntryTimestamp  int64
	ExitTimestamp   *int64
	DurationSeconds *int32
	EntryLatitude   float64
	EntryLongitude  float64
}

type visitorRepository struct {
	queries *db.Queries
}

func NewVisitorRepository(queries *db.Queries) VisitorRepository {
	return &visitorRepository{queries: queries}
}

func (r *visitorRepository) GetVisitedStreets(ctx context.Context, userID string) ([]db.VisitedStreet, error) {
	streets, err := r.queries.GetVisitedStreets(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get visited streets: %w", err)
	}
	return streets, nil
}

func (r *visitorRepository) CreateVisitedStreet(ctx context.Context, params CreateVisitedStreetParams) (*db.VisitedStreet, error) {
	sqlcParams := db.CreateVisitedStreetParams{
		UserID:         params.UserID,
		SessionID:      params.SessionID,
		StreetID:       params.StreetID,
		StreetName:     params.StreetName,
		EntryTimestamp: params.EntryTimestamp,
		ExitTimestamp: pgtype.Int8{
			Int64: int64Value(params.ExitTimestamp),
			Valid: params.ExitTimestamp != nil,
		},
		DurationSeconds: pgtype.Int4{
			Int32: int32Value(params.DurationSeconds),
			Valid: params.DurationSeconds != nil,
		},
		EntryLatitude:  decimalToNumeric(params.EntryLatitude),
		EntryLongitude: decimalToNumeric(params.EntryLongitude),
	}

	street, err := r.queries.CreateVisitedStreet(ctx, sqlcParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create visited street: %w", err)
	}
	return &street, nil
}

func (r *visitorRepository) CheckVisitedStreetExists(ctx context.Context, userID, sessionID, streetID string, entryTimestamp int64) (bool, error) {
	count, err := r.queries.CheckVisitedStreetExists(ctx, db.CheckVisitedStreetExistsParams{
		UserID:         userID,
		SessionID:      sessionID,
		StreetID:       streetID,
		EntryTimestamp: entryTimestamp,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check visited street exists: %w", err)
	}
	return count > 0, nil
}

func (r *visitorRepository) GetVisitedStreetsBySession(ctx context.Context, userID, sessionID string) ([]db.VisitedStreet, error) {
	streets, err := r.queries.GetVisitedStreetsBySession(ctx, db.GetVisitedStreetsBySessionParams{
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get visited streets by session: %w", err)
	}
	return streets, nil
}

// Helper functions
func int64Value(i *int64) int64 {
	if i != nil {
		return *i
	}
	return 0
}

func int32Value(i *int32) int32 {
	if i != nil {
		return *i
	}
	return 0
}

func decimalToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	dec := decimal.NewFromFloat(f)
	_ = n.Scan(dec.String())
	return n
}