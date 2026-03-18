package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// GraphService handles knowledge graph operations.
type GraphService struct {
	edges domain.EdgeRepository
	pages domain.PageRepository
}

// NewGraphService creates a new GraphService.
func NewGraphService(edges domain.EdgeRepository, pages domain.PageRepository) *GraphService {
	return &GraphService{edges: edges, pages: pages}
}

// GetNeighbors returns the direct graph neighbors of a page.
func (s *GraphService) GetNeighbors(ctx context.Context, pageID uuid.UUID, edgeType string, direction string) (*domain.GraphNeighbors, error) {
	center, err := s.pages.GetByID(ctx, pageID)
	if err != nil {
		return nil, err
	}

	edges, err := s.edges.ListByPage(ctx, pageID, direction, edgeType)
	if err != nil {
		return nil, err
	}

	result := &domain.GraphNeighbors{
		Center: domain.GraphNode{
			ID:              center.ID.String(),
			Title:           center.Title,
			FreshnessStatus: string(center.FreshnessStatus),
		},
		EdgeCount: len(edges),
	}

	nodeSet := map[string]bool{pageID.String(): true}
	for _, e := range edges {
		gEdge := domain.GraphEdge{
			ID:       e.ID.String(),
			EdgeType: e.EdgeType,
			Metadata: e.Metadata,
			CreatedAt: e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}

		// Enrich source and target with page titles.
		if src, err := s.pages.GetByID(ctx, e.SourcePageID); err == nil {
			gEdge.Source = domain.GraphNode{
				ID:              src.ID.String(),
				Title:           src.Title,
				FreshnessStatus: string(src.FreshnessStatus),
			}
			nodeSet[src.ID.String()] = true
		}
		if tgt, err := s.pages.GetByID(ctx, e.TargetPageID); err == nil {
			gEdge.Target = domain.GraphNode{
				ID:              tgt.ID.String(),
				Title:           tgt.Title,
				FreshnessStatus: string(tgt.FreshnessStatus),
			}
			nodeSet[tgt.ID.String()] = true
		}

		result.Edges = append(result.Edges, gEdge)
	}

	result.NodeCount = len(nodeSet)
	return result, nil
}

// CreateEdge creates a new directed graph edge.
func (s *GraphService) CreateEdge(ctx context.Context, sourceID uuid.UUID, userID string, req domain.CreateEdgeRequest) (*domain.Edge, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Validate that both pages exist.
	if _, err := s.pages.GetByID(ctx, sourceID); err != nil {
		return nil, fmt.Errorf("%w: source page not found", domain.ErrNotFound)
	}
	if _, err := s.pages.GetByID(ctx, req.TargetPageID); err != nil {
		return nil, fmt.Errorf("%w: target page not found", domain.ErrNotFound)
	}

	edge := &domain.Edge{
		SourcePageID: sourceID,
		TargetPageID: req.TargetPageID,
		EdgeType:     req.EdgeType,
		Metadata:     req.Metadata,
		CreatedBy:    uid,
	}

	return s.edges.Create(ctx, edge)
}

// Explore performs a multi-hop graph traversal from a root page.
func (s *GraphService) Explore(ctx context.Context, rootID uuid.UUID, depth int, edgeType string, limit int) (*domain.GraphExploreResult, error) {
	if _, err := s.pages.GetByID(ctx, rootID); err != nil {
		return nil, err
	}

	result, err := s.edges.Explore(ctx, rootID, depth, edgeType, limit)
	if err != nil {
		return nil, err
	}

	// Enrich nodes with page metadata.
	for i, node := range result.Nodes {
		id, parseErr := uuid.Parse(node.ID)
		if parseErr != nil {
			continue
		}
		if p, err := s.pages.GetByID(ctx, id); err == nil {
			result.Nodes[i].Title = p.Title
			result.Nodes[i].SpaceID = p.SpaceID.String()
			result.Nodes[i].FreshnessStatus = string(p.FreshnessStatus)
		}
	}

	return result, nil
}
