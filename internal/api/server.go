// Package api provides the HTTP router and server setup.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/digimon99/go2postgres/internal/api/handlers"
	"github.com/digimon99/go2postgres/internal/api/middleware"
	"github.com/digimon99/go2postgres/internal/config"
	"github.com/digimon99/go2postgres/internal/services"
	"github.com/digimon99/go2postgres/internal/static"
	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server.
type Server struct {
	cfg          *config.Config
	svc          *services.Service
	otpSvc       *services.OTPService
	handler      *handlers.Handler
	queryHandler *handlers.QueryHandler
	tableHandler *handlers.TableHandler
	router       *gin.Engine
	server       *http.Server
}

// NewServer creates a new HTTP server.
func NewServer(cfg *config.Config, svc *services.Service, otpSvc *services.OTPService) *Server {
	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		cfg:          cfg,
		svc:          svc,
		otpSvc:       otpSvc,
		handler:      handlers.NewHandler(svc, otpSvc),
		queryHandler: handlers.NewQueryHandler(svc, svc.GetPoolManager()),
		tableHandler: handlers.NewTableHandler(svc, svc.GetPoolManager()),
	}

	s.setupRouter()
	return s
}

// setupRouter configures all routes.
func (s *Server) setupRouter() {
	r := gin.New()

	// Global middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS([]string{s.cfg.FrontendURL, "http://localhost:5173", "http://localhost:3000"})) // Allow frontend origins

	// Rate limiter
	limiter := middleware.NewRateLimiter(s.cfg.RateLimitRequests, s.cfg.RateLimitWindow)

	// Health endpoints (no auth)
	r.GET("/health", s.handler.Health)
	r.GET("/ready", s.handler.Ready)

	// API v1
	v1 := r.Group("/api/v1")
	v1.Use(middleware.RateLimit(limiter))

	// Auth routes (no auth required)
	auth := v1.Group("/auth")
	{
		auth.POST("/register", s.handler.Register)
		auth.POST("/login", s.handler.Login)
		auth.POST("/refresh", s.handler.Refresh)
		auth.POST("/logout", s.handler.Logout)
		// OTP auth
		auth.POST("/otp/send", s.handler.SendOTP)
		auth.POST("/otp/verify", s.handler.VerifyOTP)
	}

	// ---- /query endpoint (API key auth, separate from JWT auth) ----
	query := v1.Group("/query")
	query.Use(middleware.APIKeyAuth(s.svc))
	{
		query.POST("", s.queryHandler.HandleQuery)
	}

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.Auth(s.svc))
	{
		// User profile
		protected.GET("/me", s.handler.GetProfile)

		// Instance routes
		instances := protected.Group("/instances")
		{
			instances.POST("", s.handler.CreateInstance)
			instances.GET("", s.handler.ListInstances)
			instances.GET("/:id", s.handler.GetInstance)
			instances.DELETE("/:id", s.handler.DeleteInstance)
			
			// Password reveal with stricter rate limit (renamed from reveal-password to avoid Cloudflare WAF)
			revealLimiter := middleware.NewRateLimiter(s.cfg.RevealPasswordLimit, time.Hour)
			instances.POST("/:id/get-db-config", middleware.RateLimit(revealLimiter), s.handler.RevealPassword)

			// API key management routes
			instances.GET("/:id/keys", s.handler.ListAPIKeys)
			instances.POST("/:id/keys", s.handler.CreateAPIKey)

			// Table editor routes
			instances.GET("/:id/tables", s.tableHandler.ListTables)
			instances.POST("/:id/tables", s.tableHandler.CreateTable)
			instances.GET("/:id/tables/:table/schema", s.tableHandler.GetTableSchema)
			instances.PATCH("/:id/tables/:table", s.tableHandler.UpdateTable)
			instances.DELETE("/:id/tables/:table", s.tableHandler.DropTable)
			instances.GET("/:id/tables/:table/rows", s.tableHandler.GetTableRows)
			instances.POST("/:id/tables/:table/rows", s.tableHandler.InsertRow)
			instances.PATCH("/:id/tables/:table/rows", s.tableHandler.UpdateRow)
			instances.DELETE("/:id/tables/:table/rows", s.tableHandler.DeleteRow)

			// SQL editor query endpoint (JWT auth for dashboard users)
			instances.POST("/:id/query", s.queryHandler.HandleDashboardQuery)
		}

		// API key revocation (by keyId across all instances)
		protected.DELETE("/keys/:keyId", s.handler.RevokeAPIKey)

		// Admin routes
		admin := protected.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			admin.GET("/stats", s.handler.AdminStats)
			admin.GET("/users", s.handler.ListAllUsers)
			admin.GET("/instances", s.handler.ListAllInstances)
			admin.POST("/users/:id/approve", s.handler.ApproveUser)
			admin.POST("/instances/:id/suspend", s.handler.SuspendInstance)
			admin.POST("/instances/:id/resume", s.handler.ResumeInstance)
		}
	}

	// Metrics endpoint (if enabled)
	if s.cfg.MetricsEnabled {
		r.GET("/metrics", s.metricsHandler())
	}

	// Serve embedded frontend static files
	// This handles all routes not matched by API handlers (SPA fallback)
	if static.HasFiles() {
		r.NoRoute(static.Handler())
	}

	s.router = r
}

// metricsHandler returns a Prometheus metrics handler.
func (s *Server) metricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement Prometheus metrics
		c.String(http.StatusOK, "# HELP go2postgres_up Server is up\n# TYPE go2postgres_up gauge\ngo2postgres_up 1\n")
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.ServerHost, s.cfg.ServerPort)
	
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start with TLS if certificates are configured
	if s.cfg.TLSCertFile != "" && s.cfg.TLSKeyFile != "" {
		return s.server.ListenAndServeTLS(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	}

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// Router returns the Gin engine for testing.
func (s *Server) Router() *gin.Engine {
	return s.router
}
