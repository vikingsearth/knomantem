package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/knomantem/knomantem/internal/domain"
)

// EdgeRepo implements domain.EdgeRepository against PostgreSQL.
type EdgeRepo struct {
	db *DB
}

// NewEdgeRepo creates a new EdgeRepo.
func NewEdgeRepo(db *DB) *EdgeRepo {
	return &EdgeRepo{db: db}
}

func (r *EdgeRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Edge, error) {
	const q = `
		SELECT id, source_page_id, target_page_id, edge_type, metadata, created_by, created_at
		FROM graph_edges WHERE id=$1`
	row := r.db.Pool.QueryRow(ctx, q, id)
	return scanEdge(row)
}

func (r *EdgeRepo) ListByPage(ctx context.Context, pageID uuid.UUID, direction string, edgeType string) ([]*domain.Edge, error) {
	var q string
	var args []any

	switch direction {
	case "outgoing":
		q = `SELECT id, source_page_id, target_page_id, edge_type, metadata, created_by, created_at
		     FROM graph_edges WHERE source_page_id=$1`
		args = []any{pageID}
	case "incoming":
		q = `SELECT id, source_page_id, target_page_id, edge_type, metadata, created_by, created_at
		     FROM graph_edges WHERE target_page_id=$1`
		args = []any{pageID}
	default:
		q = `SELECT id, source_page_id, target_page_id, edge_type, metadata, created_by, created_at
		     FROM graph_edges WHERE source_page_id=$1 OR target_page_id=$1`
		args = []any{pageID}
	}

	if edgeType != "" {
		q += " AND edge_type=$2"
		args = append(args, edgeType)
	}
	q += " ORDER BY created_at DESC"

	rows, err := r.db.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	var out []*domain.Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, mapError(rows.Err())
}

func (r *EdgeRepo) Create(ctx context.Context, e *domain.Edge) (*domain.Edge, error) {
	const q = `
		INSERT INTO graph_edges (id, source_page_id, target_page_id, edge_type, metadata, created_by)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, source_page_id, target_page_id, edge_type, metadata, created_by, created_at`
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	meta, _ := json.Marshal(e.Metadata)
	row := r.db.Pool.QueryRow(ctx, q,
		e.ID, e.SourcePageID, e.TargetPageID, e.EdgeType, meta, e.CreatedBy,
	)
	return scanEdge(row)
}

func (r *EdgeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM graph_edges WHERE id=$1`
	tag, err := r.db.Pool.Exec(ctx, q, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EdgeRepo) Explore(ctx context.Context, rootID uuid.UUID, depth int, edgeType string, limit int) (*domain.GraphExploreResult, error) {
	// Simplified BFS implementation using a recursive CTE.
	const q = `
		WITH RECURSIVE graph(source_id, target_id, edge_type, depth) AS (
			SELECT source_page_id, target_page_id, edge_type, 1
			FROM graph_edges WHERE source_page_id=$1
			UNION
			SELECT ge.source_page_id, ge.target_page_id, ge.edge_type, g.depth+1
			FROM graph_edges ge
			JOIN graph g ON ge.source_page_id = g.target_id
			WHERE g.depth < $2
		)
		SELECT source_id, target_id, edge_type FROM graph LIMIT $3`
	rows, err := r.db.Pool.Query(ctx, q, rootID, depth, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	result := &domain.GraphExploreResult{}
	nodeSet := map[string]bool{rootID.String(): true}

	for rows.Next() {
		var srcID, tgtID uuid.UUID
		var et string
		if err := rows.Scan(&srcID, &tgtID, &et); err != nil {
			return nil, mapError(err)
		}
		if edgeType == "" || edgeType == et {
			result.Edges = append(result.Edges, domain.ExploreEdge{
				SourceID: srcID.String(),
				TargetID: tgtID.String(),
				EdgeType: et,
			})
			nodeSet[srcID.String()] = true
			nodeSet[tgtID.String()] = true
		}
	}

	for id := range nodeSet {
		result.Nodes = append(result.Nodes, domain.ExploreNode{
			ID: id,
		})
	}
	result.TotalNodes = len(result.Nodes)
	result.TotalEdges = len(result.Edges)
	return result, mapError(rows.Err())
}

func scanEdge(row pgx.Row) (*domain.Edge, error) {
	e := &domain.Edge{}
	var meta []byte
	err := row.Scan(
		&e.ID, &e.SourcePageID, &e.TargetPageID, &e.EdgeType, &meta, &e.CreatedBy, &e.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &e.Metadata)
	}
	return e, nil
}
