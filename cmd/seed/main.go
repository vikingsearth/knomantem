package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://knomantem:knomantem@localhost:5432/knomantem?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	log.Println("connected to database, starting seed...")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// ------------------------------------------------------------------
	// 1. Create admin user
	// ------------------------------------------------------------------
	var adminID string
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email, display_name, password_hash, role, settings)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
		RETURNING id
	`,
		"admin@knomantem.dev",
		"Admin User",
		// bcrypt hash of "password123" (pre-computed, no runtime dependency)
		"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		"admin",
		"{}",
	).Scan(&adminID)
	if err != nil {
		log.Fatalf("failed to upsert admin user: %v", err)
	}
	log.Printf("admin user id: %s", adminID)

	// ------------------------------------------------------------------
	// 2. Create member users (5 members)
	// ------------------------------------------------------------------
	memberNames := []string{"Jane Doe", "Alex Smith", "Maria Garcia", "Liam Chen", "Sofia Patel"}
	memberEmails := []string{
		"jane@knomantem.dev",
		"alex@knomantem.dev",
		"maria@knomantem.dev",
		"liam@knomantem.dev",
		"sofia@knomantem.dev",
	}
	memberIDs := make([]string, len(memberNames))
	for i, name := range memberNames {
		var uid string
		err = pool.QueryRow(ctx, `
			INSERT INTO users (email, display_name, password_hash, role, settings)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
			RETURNING id
		`,
			memberEmails[i],
			name,
			"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			"member",
			"{}",
		).Scan(&uid)
		if err != nil {
			log.Fatalf("failed to upsert member user %s: %v", name, err)
		}
		memberIDs[i] = uid
	}
	log.Printf("created %d member users", len(memberIDs))

	allUserIDs := append([]string{adminID}, memberIDs...)

	// ------------------------------------------------------------------
	// 3. Create spaces
	// ------------------------------------------------------------------
	type spaceRow struct {
		id   string
		name string
		slug string
	}
	spaceInputs := []struct {
		name        string
		slug        string
		description string
		icon        string
		ownerIdx    int
	}{
		{"Engineering", "engineering", "Engineering team knowledge base", "rocket", 0},
		{"Design", "design", "Design system and UX guidelines", "palette", 1},
		{"Product", "product", "Product roadmaps and requirements", "map", 2},
		{"Marketing", "marketing", "Marketing campaigns and brand assets", "megaphone", 3},
		{"Operations", "operations", "Runbooks, on-call, and incident guides", "wrench", 4},
	}

	spaces := make([]spaceRow, 0, len(spaceInputs))
	for _, s := range spaceInputs {
		ownerID := allUserIDs[s.ownerIdx%len(allUserIDs)]
		var sid string
		err = pool.QueryRow(ctx, `
			INSERT INTO spaces (name, slug, description, icon, owner_id, settings)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, s.name, s.slug, s.description, s.icon, ownerID, "{}").Scan(&sid)
		if err != nil {
			log.Fatalf("failed to upsert space %s: %v", s.name, err)
		}
		spaces = append(spaces, spaceRow{id: sid, name: s.name, slug: s.slug})
	}
	log.Printf("created %d spaces", len(spaces))

	// ------------------------------------------------------------------
	// 4. Create tags
	// ------------------------------------------------------------------
	type tagRow struct {
		id    string
		name  string
		color string
	}
	tagInputs := []struct {
		name  string
		color string
	}{
		{"onboarding", "#10B981"},
		{"architecture", "#6366F1"},
		{"api", "#F59E0B"},
		{"infrastructure", "#EF4444"},
		{"devops", "#8B5CF6"},
		{"security", "#EC4899"},
		{"database", "#14B8A6"},
		{"frontend", "#F97316"},
		{"backend", "#3B82F6"},
		{"ci-cd", "#84CC16"},
		{"process", "#A855F7"},
		{"releases", "#06B6D4"},
		{"testing", "#6B7280"},
		{"performance", "#D97706"},
		{"documentation", "#0EA5E9"},
	}

	tags := make([]tagRow, 0, len(tagInputs))
	for _, t := range tagInputs {
		var tid string
		err = pool.QueryRow(ctx, `
			INSERT INTO tags (name, color, is_ai_generated)
			VALUES ($1, $2, FALSE)
			ON CONFLICT (name) DO UPDATE SET color = EXCLUDED.color
			RETURNING id
		`, t.name, t.color).Scan(&tid)
		if err != nil {
			log.Fatalf("failed to upsert tag %s: %v", t.name, err)
		}
		tags = append(tags, tagRow{id: tid, name: t.name, color: t.color})
	}
	log.Printf("created %d tags", len(tags))

	// ------------------------------------------------------------------
	// 5. Create pages (200 per space = 1000 total)
	// ------------------------------------------------------------------
	topics := []string{
		"Getting Started", "Installation Guide", "Configuration", "Deployment Guide",
		"API Reference", "Authentication", "Authorization", "Database Schema",
		"CI/CD Pipeline", "Testing Strategy", "Performance Tuning", "Security Hardening",
		"Release Process", "Incident Runbook", "On-Call Guide", "Architecture Overview",
		"Service Design", "Data Model", "GraphQL Schema", "REST Conventions",
		"Caching Strategy", "Queue Design", "Event Sourcing", "Feature Flags",
		"Observability", "Alerting Rules", "SLO Definition", "Capacity Planning",
		"Disaster Recovery", "Backup Policy", "Cost Optimization", "Secrets Management",
		"Network Architecture", "Container Strategy", "Kubernetes Setup", "Terraform Guide",
		"Code Review Process", "Branching Strategy", "Dependency Management", "Versioning Policy",
	}

	paragraphs := []string{
		"This document provides a comprehensive overview of the system and its components.",
		"Follow these steps carefully to ensure a successful setup in your environment.",
		"The architecture relies on clean separation of concerns across all service boundaries.",
		"All configuration values must be stored in environment variables, never hardcoded.",
		"Before proceeding, ensure you have reviewed the prerequisites section thoroughly.",
		"Performance benchmarks are recorded monthly and tracked in the metrics dashboard.",
		"Security considerations must be reviewed by the security team before deployment.",
		"This runbook is meant to be followed during incident response situations only.",
		"The API follows RESTful conventions with JSON request and response bodies.",
		"Automated tests cover the critical paths described in the acceptance criteria.",
		"Infrastructure changes require a review from at least two senior engineers.",
		"All database migrations must be backward-compatible with the previous version.",
		"Monitoring alerts are configured to trigger at the thresholds defined below.",
		"The deployment pipeline runs on GitHub Actions and publishes to our registry.",
		"Load testing is performed weekly to ensure performance targets are maintained.",
	}

	type pageRecord struct {
		id      string
		spaceID string
	}

	const pagesPerSpace = 200
	allPages := make([]pageRecord, 0, pagesPerSpace*len(spaces))

	for spaceIdx, sp := range spaces {
		log.Printf("seeding pages for space: %s", sp.name)

		// Create root pages (depth 0), then children (depth 1-3)
		rootCount := 10
		rootIDs := make([]string, 0, rootCount)

		for i := 0; i < rootCount; i++ {
			topic := topics[(spaceIdx*rootCount+i)%len(topics)]
			slug := fmt.Sprintf("%s-%d", slugify(topic), i)
			title := fmt.Sprintf("%s - %s", sp.name, topic)
			creatorID := allUserIDs[rng.Intn(len(allUserIDs))]
			content := buildContent(title, paragraphs, rng)

			var pid string
			err = pool.QueryRow(ctx, `
				INSERT INTO pages (space_id, parent_id, title, slug, content, position, depth, icon, is_template, created_by, updated_by)
				VALUES ($1, NULL, $2, $3, $4, $5, 0, 'page', FALSE, $6, $6)
				ON CONFLICT (space_id, slug) DO UPDATE SET title = EXCLUDED.title
				RETURNING id
			`, sp.id, title, slug, content, i, creatorID).Scan(&pid)
			if err != nil {
				log.Fatalf("failed to insert root page %s: %v", title, err)
			}
			rootIDs = append(rootIDs, pid)
			allPages = append(allPages, pageRecord{id: pid, spaceID: sp.id})
		}

		// Depth 1 children: 5 per root = 50
		depth1IDs := make([]string, 0, rootCount*5)
		for ri, rootID := range rootIDs {
			for j := 0; j < 5; j++ {
				topic := topics[(spaceIdx*50+ri*5+j)%len(topics)]
				slug := fmt.Sprintf("%s-d1-%d-%d", slugify(topic), ri, j)
				title := fmt.Sprintf("%s Overview %d.%d", topic, ri+1, j+1)
				creatorID := allUserIDs[rng.Intn(len(allUserIDs))]
				content := buildContent(title, paragraphs, rng)

				var pid string
				err = pool.QueryRow(ctx, `
					INSERT INTO pages (space_id, parent_id, title, slug, content, position, depth, icon, is_template, created_by, updated_by)
					VALUES ($1, $2, $3, $4, $5, $6, 1, 'file', FALSE, $7, $7)
					ON CONFLICT (space_id, slug) DO UPDATE SET title = EXCLUDED.title
					RETURNING id
				`, sp.id, rootID, title, slug, content, j, creatorID).Scan(&pid)
				if err != nil {
					log.Fatalf("failed to insert depth-1 page %s: %v", title, err)
				}
				depth1IDs = append(depth1IDs, pid)
				allPages = append(allPages, pageRecord{id: pid, spaceID: sp.id})
			}
		}

		// Depth 2 children: 4 per depth-1 parent = 200
		// We only have 200 per space budget; root(10) + d1(50) = 60, need 140 more at depth 2
		depth2Target := pagesPerSpace - len(rootIDs) - len(depth1IDs)
		if depth2Target < 0 {
			depth2Target = 0
		}
		for k := 0; k < depth2Target; k++ {
			parentID := depth1IDs[k%len(depth1IDs)]
			topic := topics[(spaceIdx*200+k)%len(topics)]
			slug := fmt.Sprintf("%s-d2-%d", slugify(topic), k)
			title := fmt.Sprintf("%s Detail %d", topic, k+1)
			creatorID := allUserIDs[rng.Intn(len(allUserIDs))]
			content := buildContent(title, paragraphs, rng)

			var pid string
			err = pool.QueryRow(ctx, `
				INSERT INTO pages (space_id, parent_id, title, slug, content, position, depth, icon, is_template, created_by, updated_by)
				VALUES ($1, $2, $3, $4, $5, $6, 2, 'file', FALSE, $7, $7)
				ON CONFLICT (space_id, slug) DO UPDATE SET title = EXCLUDED.title
				RETURNING id
			`, sp.id, parentID, title, slug, content, k%20, creatorID).Scan(&pid)
			if err != nil {
				log.Fatalf("failed to insert depth-2 page %s: %v", title, err)
			}
			allPages = append(allPages, pageRecord{id: pid, spaceID: sp.id})
		}
	}
	log.Printf("created %d pages total", len(allPages))

	// ------------------------------------------------------------------
	// 6. Create freshness records for all pages
	// ------------------------------------------------------------------
	statuses := []string{"fresh", "aging", "stale", "fresh", "fresh"} // weight toward fresh
	for i, pg := range allPages {
		ownerID := allUserIDs[i%len(allUserIDs)]
		status := statuses[rng.Intn(len(statuses))]
		score := freshnessScore(status, rng)
		daysAgo := rng.Intn(60)
		lastReviewed := time.Now().AddDate(0, 0, -daysAgo)
		nextReview := lastReviewed.AddDate(0, 0, 30)

		_, err = pool.Exec(ctx, `
			INSERT INTO freshness_records
				(page_id, owner_id, freshness_score, review_interval_days,
				 last_reviewed_at, next_review_at, status, decay_rate)
			VALUES ($1, $2, $3, 30, $4, $5, $6, 0.0333)
			ON CONFLICT (page_id) DO NOTHING
		`, pg.id, ownerID, score, lastReviewed, nextReview, status)
		if err != nil {
			log.Fatalf("failed to insert freshness for page %s: %v", pg.id, err)
		}
	}
	log.Printf("created freshness records for %d pages", len(allPages))

	// ------------------------------------------------------------------
	// 7. Assign tags to pages (random subset)
	// ------------------------------------------------------------------
	taggedCount := 0
	for i, pg := range allPages {
		// Tag ~70% of pages with 1-3 tags
		if rng.Float32() > 0.70 {
			continue
		}
		numTags := 1 + rng.Intn(3)
		usedTags := map[string]bool{}
		for t := 0; t < numTags; t++ {
			tag := tags[(i+t)%len(tags)]
			if usedTags[tag.id] {
				continue
			}
			usedTags[tag.id] = true
			confidence := 0.70 + rng.Float64()*0.30
			_, err = pool.Exec(ctx, `
				INSERT INTO page_tags (page_id, tag_id, confidence_score)
				VALUES ($1, $2, $3)
				ON CONFLICT (page_id, tag_id) DO NOTHING
			`, pg.id, tag.id, fmt.Sprintf("%.2f", confidence))
			if err != nil {
				log.Fatalf("failed to insert page_tag for page %s: %v", pg.id, err)
			}
			taggedCount++
		}
	}
	log.Printf("assigned %d page-tag associations", taggedCount)

	// ------------------------------------------------------------------
	// 8. Create graph edges between pages in the same space
	// ------------------------------------------------------------------
	edgeTypes := []string{"reference", "related", "depends_on", "derived_from"}
	edgeCount := 0
	// For each space, create ~50 edges among pages in that space
	pagesBySpace := map[string][]string{}
	for _, pg := range allPages {
		pagesBySpace[pg.spaceID] = append(pagesBySpace[pg.spaceID], pg.id)
	}

	for _, sp := range spaces {
		spPages := pagesBySpace[sp.id]
		if len(spPages) < 2 {
			continue
		}
		edgesForSpace := 50
		attempts := 0
		created := 0
		for created < edgesForSpace && attempts < edgesForSpace*5 {
			attempts++
			srcIdx := rng.Intn(len(spPages))
			tgtIdx := rng.Intn(len(spPages))
			if srcIdx == tgtIdx {
				continue
			}
			srcID := spPages[srcIdx]
			tgtID := spPages[tgtIdx]
			edgeType := edgeTypes[rng.Intn(len(edgeTypes))]
			creatorID := allUserIDs[rng.Intn(len(allUserIDs))]

			_, err = pool.Exec(ctx, `
				INSERT INTO graph_edges (source_page_id, target_page_id, edge_type, metadata, created_by)
				VALUES ($1, $2, $3, '{}', $4)
				ON CONFLICT (source_page_id, target_page_id, edge_type) DO NOTHING
			`, srcID, tgtID, edgeType, creatorID)
			if err != nil {
				log.Printf("warning: failed to insert graph edge: %v", err)
				continue
			}
			created++
			edgeCount++
		}
	}
	log.Printf("created %d graph edges", edgeCount)

	// ------------------------------------------------------------------
	// 9. Create permissions (grant all users access to all spaces)
	// ------------------------------------------------------------------
	permCount := 0
	for _, sp := range spaces {
		for _, uid := range allUserIDs {
			level := "viewer"
			if rng.Float32() > 0.5 {
				level = "editor"
			}
			_, err = pool.Exec(ctx, `
				INSERT INTO permissions (user_id, resource_type, resource_id, permission_level, granted_by)
				VALUES ($1, 'space', $2, $3, $4)
				ON CONFLICT (user_id, resource_type, resource_id) DO NOTHING
			`, uid, sp.id, level, adminID)
			if err != nil {
				log.Printf("warning: failed to insert permission: %v", err)
				continue
			}
			permCount++
		}
	}
	log.Printf("created %d permission records", permCount)

	log.Println("seed complete!")
	fmt.Printf("\nSeed summary:\n")
	fmt.Printf("  Users:        %d (1 admin + %d members)\n", len(allUserIDs), len(memberIDs))
	fmt.Printf("  Spaces:       %d\n", len(spaces))
	fmt.Printf("  Tags:         %d\n", len(tags))
	fmt.Printf("  Pages:        %d\n", len(allPages))
	fmt.Printf("  Graph edges:  %d\n", edgeCount)
	fmt.Printf("  Permissions:  %d\n", permCount)
	fmt.Printf("\nAdmin credentials: admin@knomantem.dev / password123\n")
}

// slugify converts a title to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// buildContent creates a minimal JSON AST document.
func buildContent(title string, paragraphs []string, rng *rand.Rand) string {
	para := paragraphs[rng.Intn(len(paragraphs))]
	para2 := paragraphs[rng.Intn(len(paragraphs))]
	return fmt.Sprintf(
		`{"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":%q}]},{"type":"paragraph","content":[{"type":"text","text":%q}]},{"type":"paragraph","content":[{"type":"text","text":%q}]}]}`,
		title, para, para2,
	)
}

// freshnessScore returns a score appropriate for the given status.
func freshnessScore(status string, rng *rand.Rand) float64 {
	switch status {
	case "fresh":
		return 70.0 + rng.Float64()*30.0 // 70–100
	case "aging":
		return 40.0 + rng.Float64()*30.0 // 40–70
	case "stale":
		return rng.Float64() * 40.0 // 0–40
	default:
		return 100.0
	}
}
