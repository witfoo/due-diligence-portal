package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/handler"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/internal/service"
	"github.com/witfoo/due-diligence-portal/pkg/envconfig"
)

// Build-time variables set via ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[portal] ")
	log.Printf("[INFO] Due Diligence Portal %s (commit=%s, built=%s)", Version, Commit, BuildTime)

	// --- Configuration ---
	dbPath := envconfig.GetEnv("DD_DB_PATH", "/data/portal.db")
	httpPort := envconfig.GetEnv("DD_HTTP_PORT", "8080")
	httpsPort := envconfig.GetEnv("DD_HTTPS_PORT", "8443")
	tlsMode := envconfig.GetEnv("DD_TLS_MODE", "self-signed")
	devMode := envconfig.GetEnvBool("DD_DEV_MODE", false)
	maxUploadSize := envconfig.GetEnvInt64("DD_MAX_UPLOAD_SIZE", 100*1024*1024)

	// --- Database ---
	db, err := repository.New(dbPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("[FATAL] Failed to run migrations: %v", err)
	}

	// --- Echo Server ---
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Client IP extraction. By default trust only the direct TCP peer so that
	// X-Forwarded-For/X-Real-IP cannot be spoofed to evade rate limiting or to
	// poison audit/NDA IP records. Enable DD_TRUST_PROXY only when running behind
	// a trusted reverse proxy that sets XFF.
	if envconfig.GetEnvBool("DD_TRUST_PROXY", false) {
		e.IPExtractor = echo.ExtractIPFromXFFHeader()
	} else {
		e.IPExtractor = echo.ExtractIPDirect()
	}

	// Global middleware.
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	// Absolute body-size backstop: large enough for the configured max upload plus
	// multipart overhead, small enough to prevent unbounded-body memory exhaustion.
	// Tighter limits are applied to the pre-auth credential endpoints below.
	e.Use(echomw.BodyLimit(fmt.Sprintf("%dB", maxUploadSize+16*1024*1024)))
	e.Use(middleware.SecurityHeaders())
	e.Use(echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogValuesFunc: func(_ echo.Context, v echomw.RequestLoggerValues) error {
			log.Printf("%s %s %d %s %s", v.Method, v.URI, v.Status, v.Latency, v.RemoteIP)
			return nil
		},
	}))

	// Rate limiting: 200 API requests per minute per IP (static assets excluded).
	rateLimit := envconfig.GetEnvInt("DD_RATE_LIMIT", 200)
	rateLimiter := middleware.NewRateLimiter(rateLimit, time.Minute)
	e.Use(rateLimiter.Middleware())

	// Stricter throttle for the unauthenticated credential endpoints to resist
	// online brute-force / credential-stuffing.
	loginLimit := envconfig.GetEnvInt("DD_LOGIN_RATE_LIMIT", 10)
	loginLimiter := middleware.NewRateLimiter(loginLimit, time.Minute)
	loginThrottle := loginLimiter.Middleware()

	// CORS is disabled by default (the UI is served same-origin). Set
	// DD_CORS_ORIGINS to a comma-separated allowlist to enable cross-origin access;
	// the wildcard "*" is never combined with the Authorization header by default.
	if corsOrigins := envconfig.GetEnvList("DD_CORS_ORIGINS", nil); len(corsOrigins) > 0 {
		e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
			AllowOrigins: corsOrigins,
			AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
			AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		}))
	}

	// --- Services ---
	jwtSecret := envconfig.GetEnv("DD_JWT_SECRET", service.DefaultJWTSecret)
	// Refuse to run with a missing, default, or weak signing secret outside dev mode:
	// HS256 is symmetric and the default value is public in source control, so using
	// it in production is a complete authentication bypass.
	if !devMode && (jwtSecret == "" || jwtSecret == service.DefaultJWTSecret || len(jwtSecret) < 32) {
		log.Fatalf("[FATAL] DD_JWT_SECRET must be set to a non-default value of at least 32 bytes " +
			"(set DD_DEV_MODE=true to bypass for local development)")
	}
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, jwtSecret)
	auditLogger := middleware.NewAuditLogger(db)

	// Ensure initial admin user exists.
	adminEmail := envconfig.GetEnv("DD_ADMIN_EMAIL", "admin@localhost")
	adminPassword := envconfig.GetEnv("DD_ADMIN_PASSWORD", "")
	generatedPw, err := authSvc.EnsureAdminExists(context.Background(), adminEmail, adminPassword)
	if err != nil {
		log.Printf("[WARN] Failed to ensure admin exists: %v", err)
	}
	if generatedPw != "" {
		// Never log the credential. Write it once to a restricted file so it is not
		// persisted to the application/Docker log stream (CWE-532).
		pwFile := envconfig.GetEnv("DD_ADMIN_PASSWORD_FILE", filepath.Join(filepath.Dir(dbPath), "initial-admin-password.txt"))
		if werr := os.WriteFile(pwFile, []byte(generatedPw+"\n"), 0o600); werr != nil {
			log.Printf("[WARN] Generated an initial admin password but could not write %s: %v "+
				"(set DD_ADMIN_PASSWORD to provide one explicitly)", pwFile, werr)
		} else {
			log.Printf("[INFO] Generated initial admin password written to %s (mode 0600). "+
				"Log in and change it immediately, then delete the file.", pwFile)
		}
	}

	// --- Auth Middleware ---
	authMW := middleware.JWTAuth(authSvc)
	adminOnly := middleware.RequireRole(domain.RoleAdmin)

	// --- Repositories ---
	docRepo := repository.NewDocumentRepository(db)
	catRepo := repository.NewCategoryRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	qaRepo := repository.NewQARepository(db)
	auditRepo := repository.NewAuditRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	brandingRepo := repository.NewBrandingRepository(db)
	ndaRepo := repository.NewNDARepository(db)
	wmRepo := repository.NewWatermarkRepository(db)

	// --- Email Service ---
	emailSvc := service.NewEmailService()
	if emailSvc.IsEnabled() {
		log.Printf("[INFO] Email notifications enabled via %s", envconfig.GetEnv("DD_SMTP_HOST", ""))
	} else {
		log.Printf("[INFO] Email notifications disabled (set DD_SMTP_ENABLED=true to enable)")
	}

	// --- Register Handlers ---
	healthHandler := handler.NewHealthHandler(db, Version, Commit, BuildTime)
	healthHandler.RegisterRoutes(e)

	authHandler := handler.NewAuthHandler(authSvc, auditLogger)
	authHandler.RegisterRoutes(e, authMW, loginThrottle)

	// Route groups.
	adminGroup := e.Group("/api/v1", authMW, adminOnly)
	authGroup := e.Group("/api/v1", authMW)

	// User management (admin only).
	userHandler := handler.NewUserHandler(userRepo, authSvc, emailSvc, auditLogger)
	userHandler.RegisterRoutes(adminGroup)

	// Documents (all handlers register on authGroup; internal role checks as needed).
	docHandler := handler.NewDocumentHandler(docRepo, permRepo, auditLogger)
	docHandler.RegisterRoutes(authGroup)

	// Categories.
	catHandler := handler.NewCategoryHandler(catRepo, docRepo, auditLogger)
	catHandler.RegisterRoutes(authGroup)

	// Permissions (admin only).
	permHandler := handler.NewPermissionHandler(permRepo, auditLogger)
	permHandler.RegisterRoutes(adminGroup)

	// Q&A.
	qaHandler := handler.NewQAHandler(qaRepo, permRepo, userRepo, emailSvc, auditLogger)
	qaHandler.RegisterRoutes(authGroup)

	// Audit log (admin only).
	auditHandler := handler.NewAuditHandler(auditRepo, auditLogger)
	auditHandler.RegisterRoutes(adminGroup)

	// Analytics.
	analyticsHandler := handler.NewAnalyticsHandler(analyticsRepo, docRepo, permRepo, userRepo, auditLogger)
	analyticsHandler.RegisterRoutes(authGroup)

	// Branding.
	brandingHandler := handler.NewBrandingHandler(brandingRepo, auditLogger)
	brandingHandler.RegisterRoutes(authGroup)

	// NDA.
	ndaHandler := handler.NewNDAHandler(ndaRepo, emailSvc, adminEmail, auditLogger)
	ndaHandler.RegisterRoutes(authGroup)

	// Watermark (admin only).
	wmHandler := handler.NewWatermarkHandler(wmRepo, auditLogger)
	wmHandler.RegisterRoutes(authGroup)

	// --- Static file serving (embedded UI) ---
	setupStaticFiles(e)

	// --- Graceful Shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Start Server ---
	go func() {
		if err := startServer(e, tlsMode, httpPort, httpsPort); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("[INFO] Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] Shutdown error: %v", err)
	}
	log.Printf("[INFO] Server stopped")
}

func startServer(e *echo.Echo, tlsMode, httpPort, httpsPort string) error {
	switch tlsMode {
	case "none":
		log.Printf("[INFO] Starting HTTP server on :%s (TLS disabled)", httpPort)
		return e.Start(":" + httpPort)

	case "custom":
		certPath := envconfig.GetEnv("DD_TLS_CERT_PATH", "/certs/tls.crt")
		keyPath := envconfig.GetEnv("DD_TLS_KEY_PATH", "/certs/tls.key")
		log.Printf("[INFO] Starting HTTPS server on :%s with custom certificate", httpsPort)
		go startHTTPRedirect(httpPort, httpsPort)
		return e.StartTLS(":"+httpsPort, certPath, keyPath)

	case "self-signed":
		certDir := envconfig.GetEnv("DD_CERT_DIR", "/data/certs")
		certPath, keyPath, err := ensureSelfSignedCert(certDir)
		if err != nil {
			return fmt.Errorf("generate self-signed cert: %w", err)
		}
		log.Printf("[INFO] Starting HTTPS server on :%s with self-signed certificate", httpsPort)
		go startHTTPRedirect(httpPort, httpsPort)
		return e.StartTLS(":"+httpsPort, certPath, keyPath)

	default:
		return fmt.Errorf("unknown TLS mode: %s (expected: none, custom, self-signed)", tlsMode)
	}
}

// startHTTPRedirect starts a background HTTP server that redirects to HTTPS
// and serves health checks for Docker HEALTHCHECK.
func startHTTPRedirect(httpPort, httpsPort string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		target := fmt.Sprintf("https://%s:%s%s", host, httpsPort, r.URL.RequestURI())
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
	log.Printf("[INFO] Starting HTTP redirect on :%s -> :%s", httpPort, httpsPort)
	srv := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("[WARN] HTTP redirect server error: %v", err)
	}
}

// ensureSelfSignedCert generates a self-signed TLS certificate if one doesn't exist.
func ensureSelfSignedCert(certDir string) (certPath, keyPath string, err error) {
	certPath = filepath.Join(certDir, "tls.crt")
	keyPath = filepath.Join(certDir, "tls.key")

	// Check if cert already exists.
	if _, err := os.Stat(certPath); err == nil {
		log.Printf("[INFO] Using existing self-signed certificate: %s", certPath)
		return certPath, keyPath, nil
	}

	log.Printf("[INFO] Generating self-signed TLS certificate...")

	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create cert directory: %w", err)
	}

	// Generate ECDSA key.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}

	// Create certificate template.
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("generate serial number: %w", err)
	}

	// Populate the Subject CN and SANs from DD_TLS_HOSTNAME so the self-signed cert
	// is valid for the actual deployment host, not just localhost.
	hostname := envconfig.GetEnv("DD_TLS_HOSTNAME", "localhost")
	dnsNames := []string{"localhost"}
	var ipAddrs []net.IP
	if hostname != "" && hostname != "localhost" {
		if ip := net.ParseIP(hostname); ip != nil {
			ipAddrs = append(ipAddrs, ip)
		} else {
			dnsNames = append(dnsNames, hostname)
		}
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Due Diligence Portal"},
			CommonName:   hostname,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddrs,
	}

	// Create certificate.
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", "", fmt.Errorf("create certificate: %w", err)
	}

	// Write certificate file.
	if err := writePEMFile(certPath, "CERTIFICATE", certDER, 0o644); err != nil {
		return "", "", fmt.Errorf("write cert: %w", err)
	}

	// Write key file.
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", fmt.Errorf("marshal key: %w", err)
	}
	if err := writePEMFile(keyPath, "EC PRIVATE KEY", keyDER, 0o600); err != nil {
		return "", "", fmt.Errorf("write key: %w", err)
	}

	log.Printf("[INFO] Self-signed certificate generated: %s", certPath)
	return certPath, keyPath, nil
}

// setupStaticFiles serves the pre-built SvelteKit UI from the ui/build directory.
// In production, the UI is embedded in the binary. In development, it falls back
// to serving from the filesystem.
func setupStaticFiles(e *echo.Echo) {
	uiBuildPath := envconfig.GetEnv("DD_UI_PATH", "ui/build")

	// Check if the UI build directory exists on the filesystem.
	if info, err := os.Stat(uiBuildPath); err == nil && info.IsDir() {
		log.Printf("[INFO] Serving UI from filesystem: %s", uiBuildPath)

		// Serve static files with SPA fallback.
		// spaFileHandler uses http.Dir which handles path sanitization internally.
		spaHandler := newSPAFileHandler(http.Dir(uiBuildPath))
		e.GET("/*", echo.WrapHandler(spaHandler))
		return
	}

	// Fallback: try embedded FS (will be available in production builds).
	log.Printf("[INFO] No UI build directory found at %s, serving placeholder", uiBuildPath)
	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, `<!DOCTYPE html>
<html><head><title>Due Diligence Portal</title></head>
<body><h1>Due Diligence Portal</h1><p>UI not built. Run <code>cd ui && npm run build</code></p></body>
</html>`)
	})
}

// writePEMFile writes a PEM-encoded block to a file, ensuring the file is
// properly closed after writing (handles the errcheck/unhandled-close issue).
func writePEMFile(path, blockType string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	encodeErr := pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
	closeErr := f.Close()
	if encodeErr != nil {
		return fmt.Errorf("encode PEM to %s: %w", path, encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", path, closeErr)
	}
	return nil
}

// spaFileHandler serves static files from an http.FileSystem with SPA fallback.
// If the requested file doesn't exist, it serves index.html for client-side routing.
// Uses http.Dir which handles path sanitization internally (no user-controlled os.Stat).
type spaFileHandler struct {
	fs      http.FileSystem
	handler http.Handler
}

func newSPAFileHandler(fs http.FileSystem) *spaFileHandler {
	return &spaFileHandler{fs: fs, handler: http.FileServer(fs)}
}

func (h *spaFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path

	// Don't serve static files for API/health routes.
	if strings.HasPrefix(urlPath, "/api/") || urlPath == "/health" || urlPath == "/ready" || urlPath == "/version" {
		http.NotFound(w, r)
		return
	}

	// Try to open the file via http.Dir (handles path sanitization).
	f, err := h.fs.Open(urlPath)
	if err != nil {
		// File not found — serve SPA fallback (index.html).
		r.URL.Path = "/"
		h.handler.ServeHTTP(w, r)
		return
	}
	f.Close()

	// File exists — serve it directly.
	h.handler.ServeHTTP(w, r)
}

// Ensure fs package is used (needed for future embed.FS usage).
var _ fs.FS
