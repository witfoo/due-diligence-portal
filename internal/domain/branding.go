package domain

import "time"

// BrandingConfig holds the white-label configuration.
type BrandingConfig struct {
	ID                 string    `json:"id"`
	CompanyName        string    `json:"company_name"`
	PrimaryColor       string    `json:"primary_color"`
	SecondaryColor     string    `json:"secondary_color"`
	AccentColor        string    `json:"accent_color"`
	ErrorColor         string    `json:"error_color"`
	WarningColor       string    `json:"warning_color"`
	SuccessColor       string    `json:"success_color"`
	InfoColor          string    `json:"info_color"`
	BackgroundColor    string    `json:"background_color"`
	SurfaceColor       string    `json:"surface_color"`
	TextColor          string    `json:"text_color"`
	TextSecondaryColor string    `json:"text_secondary_color"`
	BorderColor        string    `json:"border_color"`
	HoverColor         string    `json:"hover_color"`
	ActiveColor        string    `json:"active_color"`
	HeaderColor        string    `json:"header_color"`
	SidebarColor       string    `json:"sidebar_color"`
	FontFamily         string    `json:"font_family,omitempty"`
	CustomCSS          string    `json:"custom_css,omitempty"`
	DocumentTitle      string    `json:"document_title,omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ValidAssetKeys contains the allowed branding asset keys.
var ValidAssetKeys = map[string]bool{
	"logo":             true,
	"logo_dark":        true,
	"favicon":          true,
	"login_background": true,
	"email_header":     true,
	"report_header":    true,
	"report_footer":    true,
}

// BrandingAsset holds a binary branding asset (logo, favicon, etc.).
type BrandingAsset struct {
	Key            string    `json:"key"`
	FileData       []byte    `json:"-"`
	MimeType       string    `json:"mime_type"`
	FileSize       int64     `json:"file_size"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
	UploadedBy     string    `json:"uploaded_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// DefaultBrandingConfig returns the default branding configuration.
func DefaultBrandingConfig() BrandingConfig {
	return BrandingConfig{
		ID:                 "default",
		CompanyName:        "Company",
		PrimaryColor:       "#0f62fe",
		SecondaryColor:     "#393939",
		AccentColor:        "#f1c21b",
		ErrorColor:         "#da1e28",
		WarningColor:       "#f1c21b",
		SuccessColor:       "#24a148",
		InfoColor:          "#4589ff",
		BackgroundColor:    "#161616",
		SurfaceColor:       "#262626",
		TextColor:          "#f4f4f4",
		TextSecondaryColor: "#c6c6c6",
		BorderColor:        "#393939",
		HoverColor:         "#353535",
		ActiveColor:        "#525252",
		HeaderColor:        "#161616",
		SidebarColor:       "#1c1c1c",
	}
}
