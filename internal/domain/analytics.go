package domain

import "time"

// ViewEvent records a user viewing a document.
type ViewEvent struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	DocumentID string    `json:"document_id"`
	DurationMs *int64    `json:"duration_ms,omitempty"`
	PageCount  *int      `json:"page_count,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// EngagementSummary provides aggregated analytics for the dashboard.
type EngagementSummary struct {
	TotalDocuments    int `json:"total_documents"`
	TotalViews        int `json:"total_views"`
	TotalDownloads    int `json:"total_downloads"`
	UniqueViewers     int `json:"unique_viewers"`
	ActiveInvestors   int `json:"active_investors"`
	OpenQuestions     int `json:"open_questions"`
	PendingNDAs       int `json:"pending_ndas"`
	RecentViewCount   int `json:"recent_view_count"`
}

// DocumentAnalytics provides per-document engagement data.
type DocumentAnalytics struct {
	DocumentID   string `json:"document_id"`
	DocumentName string `json:"document_name"`
	ViewCount    int    `json:"view_count"`
	UniqueViewers int   `json:"unique_viewers"`
	DownloadCount int   `json:"download_count"`
	AvgDurationMs int64 `json:"avg_duration_ms"`
	LastViewedAt  *time.Time `json:"last_viewed_at,omitempty"`
}

// UserAnalytics provides per-user engagement data.
type UserAnalytics struct {
	UserID          string     `json:"user_id"`
	UserName        string     `json:"user_name"`
	UserEmail       string     `json:"user_email"`
	DocumentsViewed int        `json:"documents_viewed"`
	TotalViews      int        `json:"total_views"`
	TotalDurationMs int64      `json:"total_duration_ms"`
	LastActiveAt    *time.Time `json:"last_active_at,omitempty"`
}

// WatermarkConfig holds the watermark configuration.
type WatermarkConfig struct {
	ID           string  `json:"id"`
	Enabled      bool    `json:"enabled"`
	TextTemplate string  `json:"text_template"`
	Position     string  `json:"position"`
	Opacity      float64 `json:"opacity"`
	FontSize     int     `json:"font_size"`
	Color        string  `json:"color"`
	UpdatedAt    time.Time `json:"updated_at"`
}
