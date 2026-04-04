package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// AnalyticsRepository defines the interface for analytics data access.
type AnalyticsRepository interface {
	RecordView(ctx context.Context, event *domain.ViewEvent) error
	GetDocumentAnalytics(ctx context.Context, documentID string) (*domain.DocumentAnalytics, error)
	GetUserAnalytics(ctx context.Context, userID string) (*domain.UserAnalytics, error)
	GetEngagementSummary(ctx context.Context) (*domain.EngagementSummary, error)
}

type analyticsRepository struct {
	db *DB
}

// NewAnalyticsRepository creates a new AnalyticsRepository backed by SQLite.
func NewAnalyticsRepository(db *DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) RecordView(ctx context.Context, event *domain.ViewEvent) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO view_events (id, user_id, document_id, duration_ms, page_count, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.ID, event.UserID, event.DocumentID, event.DurationMs, event.PageCount, now,
	)
	if err != nil {
		return fmt.Errorf("record view event: %w", err)
	}
	event.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *analyticsRepository) GetDocumentAnalytics(ctx context.Context, documentID string) (*domain.DocumentAnalytics, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT
			v.document_id,
			COALESCE(d.name, ''),
			COUNT(*) as view_count,
			COUNT(DISTINCT v.user_id) as unique_viewers,
			COALESCE(AVG(v.duration_ms), 0) as avg_duration_ms,
			MAX(v.created_at) as last_viewed_at
		 FROM view_events v
		 LEFT JOIN documents d ON d.id = v.document_id
		 WHERE v.document_id = ?
		 GROUP BY v.document_id`, documentID)

	var da domain.DocumentAnalytics
	var lastViewedAt sql.NullString

	err := row.Scan(&da.DocumentID, &da.DocumentName, &da.ViewCount,
		&da.UniqueViewers, &da.AvgDurationMs, &lastViewedAt)
	if err == sql.ErrNoRows {
		return &domain.DocumentAnalytics{DocumentID: documentID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get document analytics: %w", err)
	}

	if lastViewedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastViewedAt.String)
		da.LastViewedAt = &t
	}
	return &da, nil
}

func (r *analyticsRepository) GetUserAnalytics(ctx context.Context, userID string) (*domain.UserAnalytics, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT
			v.user_id,
			COALESCE(u.name, ''),
			COALESCE(u.email, ''),
			COUNT(DISTINCT v.document_id) as documents_viewed,
			COUNT(*) as total_views,
			COALESCE(SUM(v.duration_ms), 0) as total_duration_ms,
			MAX(v.created_at) as last_active_at
		 FROM view_events v
		 LEFT JOIN users u ON u.id = v.user_id
		 WHERE v.user_id = ?
		 GROUP BY v.user_id`, userID)

	var ua domain.UserAnalytics
	var lastActiveAt sql.NullString

	err := row.Scan(&ua.UserID, &ua.UserName, &ua.UserEmail,
		&ua.DocumentsViewed, &ua.TotalViews, &ua.TotalDurationMs, &lastActiveAt)
	if err == sql.ErrNoRows {
		return &domain.UserAnalytics{UserID: userID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user analytics: %w", err)
	}

	if lastActiveAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastActiveAt.String)
		ua.LastActiveAt = &t
	}
	return &ua, nil
}

func (r *analyticsRepository) GetEngagementSummary(ctx context.Context) (*domain.EngagementSummary, error) {
	var s domain.EngagementSummary

	// Total documents.
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM documents`).Scan(&s.TotalDocuments); err != nil {
		return nil, fmt.Errorf("engagement summary (documents): %w", err)
	}

	// Total views.
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM view_events`).Scan(&s.TotalViews); err != nil {
		return nil, fmt.Errorf("engagement summary (views): %w", err)
	}

	// Unique viewers.
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM view_events`).Scan(&s.UniqueViewers); err != nil {
		return nil, fmt.Errorf("engagement summary (unique viewers): %w", err)
	}

	// Active investors (viewed something in last 30 days).
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT v.user_id) FROM view_events v
		 JOIN users u ON u.id = v.user_id
		 WHERE u.role = 'investor'
		 AND v.created_at >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-30 days')`).Scan(&s.ActiveInvestors); err != nil {
		return nil, fmt.Errorf("engagement summary (active investors): %w", err)
	}

	// Open questions.
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM qa_threads WHERE status = 'open'`).Scan(&s.OpenQuestions); err != nil {
		return nil, fmt.Errorf("engagement summary (open questions): %w", err)
	}

	// Recent views (last 7 days).
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM view_events
		 WHERE created_at >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-7 days')`).Scan(&s.RecentViewCount); err != nil {
		return nil, fmt.Errorf("engagement summary (recent views): %w", err)
	}

	// Total downloads from audit log.
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE action = 'document.downloaded'`).Scan(&s.TotalDownloads); err != nil {
		return nil, fmt.Errorf("engagement summary (downloads): %w", err)
	}

	return &s, nil
}
