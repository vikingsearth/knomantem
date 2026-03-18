package domain

import (
	"time"

	"github.com/google/uuid"
)

// Edge represents a directed knowledge-graph edge between two Pages.
type Edge struct {
	ID           uuid.UUID      `json:"id"`
	SourcePageID uuid.UUID      `json:"source_page_id"`
	TargetPageID uuid.UUID      `json:"target_page_id"`
	EdgeType     string         `json:"edge_type"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedBy    uuid.UUID      `json:"created_by"`
	CreatedAt    time.Time      `json:"created_at"`
}

// CreateEdgeRequest carries the fields needed to create a new Edge.
type CreateEdgeRequest struct {
	TargetPageID uuid.UUID      `json:"target_page_id"`
	EdgeType     string         `json:"edge_type"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// GraphNeighbors is the response for the GET /pages/:id/graph endpoint.
type GraphNeighbors struct {
	Center    GraphNode   `json:"center"`
	Edges     []GraphEdge `json:"edges"`
	NodeCount int         `json:"node_count"`
	EdgeCount int         `json:"edge_count"`
}

// GraphNode is a minimal page node used in graph responses.
type GraphNode struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	FreshnessStatus string `json:"freshness_status"`
}

// GraphEdge is an edge in a graph response.
type GraphEdge struct {
	ID        string         `json:"id"`
	Source    GraphNode      `json:"source"`
	Target    GraphNode      `json:"target"`
	EdgeType  string         `json:"edge_type"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// GraphExploreResult is the response for GET /graph/explore.
type GraphExploreResult struct {
	Nodes      []ExploreNode `json:"nodes"`
	Edges      []ExploreEdge `json:"edges"`
	TotalNodes int           `json:"total_nodes"`
	TotalEdges int           `json:"total_edges"`
	Truncated  bool          `json:"truncated"`
}

// ExploreNode is a node in the exploration subgraph.
type ExploreNode struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	SpaceID         string `json:"space_id"`
	FreshnessStatus string `json:"freshness_status"`
	ConnectionCount int    `json:"connection_count"`
	DepthFromRoot   int    `json:"depth_from_root"`
}

// ExploreEdge is an edge in the exploration subgraph.
type ExploreEdge struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	EdgeType string `json:"edge_type"`
}
