package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/ghupdate"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"

	"github.com/tigawanna/pockestrator/internal/caddy"
	"github.com/tigawanna/pockestrator/internal/database"
	"github.com/tigawanna/pockestrator/internal/service"
	"github.com/tigawanna/pockestrator/internal/systemd"
	"github.com/tigawanna/pockestrator/internal/validation"
	"github.com/tigawanna/pockestrator/pkg"
)

//go:embed dashboard/dist/*
var dashboardFiles embed.FS

// PocketstratorApp represents the main application
type PocketstratorApp struct {
	app          *pocketbase.PocketBase
	orchestrator *pkg.Orchestrator
	config       *Config
}

// Config holds application configuration
type Config struct {
	BaseDir       string
	SystemdDir    string
	CaddyConfig   string
	DefaultDomain string
	PublicDir     string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		BaseDir:       "/home/ubuntu",
		SystemdDir:    "/lib/systemd/system",
		CaddyConfig:   "/etc/caddy/Caddyfile",
		DefaultDomain: "tigawanna.vip",
		PublicDir:     defaultPublicDir(),
	}
}

func main() {
	app := pocketbase.New()
	config := DefaultConfig()

	// Initialize managers
	serviceManager := service.NewManager(config.BaseDir, config.SystemdDir, config.CaddyConfig)
	systemdManager := systemd.NewManager(config.SystemdDir)
	caddyManager := caddy.NewManager(config.CaddyConfig)
	validator := validation.NewValidator(config.BaseDir, config.SystemdDir, config.CaddyConfig)
	dbManager := database.NewManager(app)

	// Initialize orchestrator
	orchestratorConfig := &pkg.Config{
		BaseDir:       config.BaseDir,
		SystemdDir:    config.SystemdDir,
		CaddyConfig:   config.CaddyConfig,
		DefaultDomain: config.DefaultDomain,
	}

	orchestrator := pkg.NewOrchestrator(
		serviceManager,
		systemdManager,
		caddyManager,
		validator,
		dbManager,
		orchestratorConfig,
	)

	pockApp := &PocketstratorApp{
		app:          app,
		orchestrator: orchestrator,
		config:       config,
	}

	// Setup command line flags
	setupFlags(app, config)

	// Setup plugins
	setupPlugins(app, config)

	// Setup hooks and routes
	pockApp.setupHooks()
	pockApp.setupRoutes()

	// Setup static file serving
	setupStaticFiles(app, config.PublicDir)

	log.Println("🚀 Pockestrator starting...")
	log.Printf("📁 Base directory: %s", config.BaseDir)
	log.Printf("⚙️  SystemD directory: %s", config.SystemdDir)
	log.Printf("🌐 Caddy config: %s", config.CaddyConfig)
	log.Printf("🏠 Default domain: %s", config.DefaultDomain)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// setupFlags sets up command line flags
func setupFlags(app *pocketbase.PocketBase, config *Config) {
	var hooksDir string
	app.RootCmd.PersistentFlags().StringVar(
		&hooksDir,
		"hooksDir",
		"",
		"the directory with the JS app hooks",
	)

	var hooksWatch bool
	app.RootCmd.PersistentFlags().BoolVar(
		&hooksWatch,
		"hooksWatch",
		true,
		"auto restart the app on pb_hooks file change",
	)

	var hooksPool int
	app.RootCmd.PersistentFlags().IntVar(
		&hooksPool,
		"hooksPool",
		15,
		"the total prewarm goja.Runtime instances for the JS app hooks execution",
	)

	var migrationsDir string
	app.RootCmd.PersistentFlags().StringVar(
		&migrationsDir,
		"migrationsDir",
		"./migrations",
		"the directory with the user defined migrations",
	)

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(
		&automigrate,
		"automigrate",
		true,
		"enable/disable auto migrations",
	)

	var indexFallback bool
	app.RootCmd.PersistentFlags().BoolVar(
		&indexFallback,
		"indexFallback",
		true,
		"fallback the request to index.html on missing static path",
	)

	app.RootCmd.ParseFlags(os.Args[1:])
}

// setupPlugins sets up PocketBase plugins
func setupPlugins(app *pocketbase.PocketBase, config *Config) {
	// load jsvm (pb_hooks and pb_migrations)
	jsvm.MustRegister(app, jsvm.Config{
		MigrationsDir: "./migrations",
		HooksWatch:    true,
		HooksPoolSize: 15,
	})

	// migrate command (with js templates)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  true,
		Dir:          "./migrations",
	})

	// GitHub selfupdate
	ghupdate.MustRegister(app, app.RootCmd, ghupdate.Config{})
}

// setupStaticFiles sets up static file serving
func setupStaticFiles(app *pocketbase.PocketBase, publicDir string) {
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				// Try to serve from embedded dashboard files first
				dashboardFS, err := fs.Sub(dashboardFiles, "dashboard/dist")
				if err == nil {
					e.Router.GET("/{path...}", apis.Static(dashboardFS, true))
				} else {
					// Fallback to local filesystem
					e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), true))
				}
			}
			return e.Next()
		},
		Priority: 999,
	})
}

// setupHooks sets up PocketBase event hooks
func (p *PocketstratorApp) setupHooks() {
	// Service creation hook
	p.app.OnRecordCreate("services").BindFunc(func(e *core.RecordEvent) error {
		return p.handleServiceCreate(e)
	})

	// Service update hook
	p.app.OnRecordUpdate("services").BindFunc(func(e *core.RecordEvent) error {
		return p.handleServiceUpdate(e)
	})

	// Service deletion hook
	p.app.OnRecordDelete("services").BindFunc(func(e *core.RecordEvent) error {
		return p.handleServiceDelete(e)
	})

	// Health check job - runs every 5 minutes
	p.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Schedule periodic health checks
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					p.performHealthCheck()
				}
			}
		}()
		return e.Next()
	})

	// App startup hook
	p.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		log.Println("✅ Pockestrator is ready!")
		log.Printf("🌐 Admin UI: http://localhost:%s/_/", e.App.Settings().Meta.AppURL)
		return e.Next()
	})
}

// setupRoutes sets up custom API routes
func (p *PocketstratorApp) setupRoutes() {
	p.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Service management endpoints
		e.Router.POST("/api/pockestrator/services", p.handleCreateService)
		e.Router.GET("/api/pockestrator/services", p.handleListServices)
		e.Router.GET("/api/pockestrator/services/{id}", p.handleGetService)
		e.Router.DELETE("/api/pockestrator/services/{id}", p.handleDeleteService)
		e.Router.POST("/api/pockestrator/services/{id}/control", p.handleServiceControl)
		e.Router.GET("/api/pockestrator/services/{id}/status", p.handleServiceStatus)
		e.Router.GET("/api/pockestrator/services/{id}/logs", p.handleServiceLogs)

		// Validation endpoints
		e.Router.POST("/api/pockestrator/validate/service", p.handleValidateService)
		e.Router.GET("/api/pockestrator/validate/system", p.handleValidateSystem)

		// System information endpoints
		e.Router.GET("/api/pockestrator/system/info", p.handleSystemInfo)
		e.Router.GET("/api/pockestrator/system/health", p.handleSystemHealth)

		return e.Next()
	})
}

// Event Handlers
func (p *PocketstratorApp) handleServiceCreate(e *core.RecordEvent) error {
	log.Printf("📦 Creating service: %s", e.Record.GetString("project_name"))
	e.Record.Set("status", "deploying")
	return nil
}

func (p *PocketstratorApp) handleServiceUpdate(e *core.RecordEvent) error {
	log.Printf("🔄 Updating service: %s", e.Record.GetString("project_name"))
	return nil
}

func (p *PocketstratorApp) handleServiceDelete(e *core.RecordEvent) error {
	ctx := context.Background()
	projectName := e.Record.GetString("project_name")

	log.Printf("🗑️  Deleting service: %s", projectName)

	// Use orchestrator to handle the deletion
	_, err := p.orchestrator.DeleteService(ctx, e.Record.Id)
	if err != nil {
		log.Printf("❌ Failed to delete service %s: %v", projectName, err)
		return fmt.Errorf("service deletion failed: %w", err)
	}

	log.Printf("✅ Service %s deleted successfully", projectName)
	return nil
}

// API Handlers
func (p *PocketstratorApp) handleCreateService(e *core.RequestEvent) error {
	ctx := context.Background()

	var req pkg.ServiceRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	// Set the creator
	if auth := e.Auth; auth != nil {
		req.CreatedBy = auth.Email()
	}

	response, err := p.orchestrator.CreateService(ctx, &req)
	if err != nil {
		return e.InternalServerError("Failed to create service", err)
	}

	return e.JSON(200, response)
}

func (p *PocketstratorApp) handleListServices(e *core.RequestEvent) error {
	ctx := context.Background()

	services, err := p.orchestrator.ListServices(ctx)
	if err != nil {
		return e.InternalServerError("Failed to list services", err)
	}

	return e.JSON(200, map[string]any{
		"services": services,
		"total":    len(services),
	})
}

func (p *PocketstratorApp) handleGetService(e *core.RequestEvent) error {
	ctx := context.Background()
	id := e.Request.PathValue("id")

	response, err := p.orchestrator.GetService(ctx, id)
	if err != nil {
		return e.NotFoundError("Service not found", err)
	}

	return e.JSON(200, response)
}

func (p *PocketstratorApp) handleDeleteService(e *core.RequestEvent) error {
	ctx := context.Background()
	id := e.Request.PathValue("id")

	response, err := p.orchestrator.DeleteService(ctx, id)
	if err != nil {
		return e.InternalServerError("Failed to delete service", err)
	}

	return e.JSON(200, response)
}

func (p *PocketstratorApp) handleServiceControl(e *core.RequestEvent) error {
	ctx := context.Background()
	id := e.Request.PathValue("id")

	var req struct {
		Action string `json:"action"`
	}

	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	response, err := p.orchestrator.ControlService(ctx, id, req.Action)
	if err != nil {
		return e.InternalServerError("Failed to control service", err)
	}

	return e.JSON(200, response)
}

func (p *PocketstratorApp) handleServiceStatus(e *core.RequestEvent) error {
	ctx := context.Background()
	id := e.Request.PathValue("id")

	status, err := p.orchestrator.GetServiceStatus(ctx, id)
	if err != nil {
		return e.InternalServerError("Failed to get service status", err)
	}

	return e.JSON(200, status)
}

func (p *PocketstratorApp) handleServiceLogs(e *core.RequestEvent) error {
	ctx := context.Background()
	id := e.Request.PathValue("id")

	// Get query parameters
	lines := 100 // default
	if linesParam := e.Request.URL.Query().Get("lines"); linesParam != "" {
		if parsedLines, err := strconv.Atoi(linesParam); err == nil && parsedLines > 0 {
			lines = parsedLines
		}
	}

	response, err := p.orchestrator.GetServiceLogs(ctx, id, lines)
	if err != nil {
		return e.InternalServerError("Failed to get service logs", err)
	}

	return e.JSON(200, response)
}

func (p *PocketstratorApp) handleValidateService(e *core.RequestEvent) error {
	ctx := context.Background()

	var req pkg.ServiceRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	// Get existing services and used ports for validation
	existingServices, err := p.orchestrator.ListServices(ctx)
	if err != nil {
		return e.InternalServerError("Failed to get existing services", err)
	}

	usedPorts, err := p.orchestrator.GetUsedPorts(ctx)
	if err != nil {
		return e.InternalServerError("Failed to get used ports", err)
	}

	// Perform validation
	validationResult := p.orchestrator.ValidateServiceConfiguration(
		req.ProjectName,
		req.Port,
		req.PocketBaseVersion,
		req.Domain,
		existingServices,
		usedPorts,
	)

	statusCode := 200
	if !validationResult.IsValid {
		statusCode = 400
	}

	return e.JSON(statusCode, map[string]any{
		"valid":    validationResult.IsValid,
		"errors":   validationResult.Errors,
		"warnings": validationResult.Warnings,
	})
}

func (p *PocketstratorApp) handleValidateSystem(e *core.RequestEvent) error {
	result := p.orchestrator.ValidateSystemRequirements()
	return e.JSON(200, result)
}

func (p *PocketstratorApp) handleSystemInfo(e *core.RequestEvent) error {
	return e.JSON(200, map[string]any{
		"version": "1.0.0",
		"uptime":  time.Since(time.Now()).String(),
		"config": map[string]any{
			"base_dir":       p.config.BaseDir,
			"systemd_dir":    p.config.SystemdDir,
			"caddy_config":   p.config.CaddyConfig,
			"default_domain": p.config.DefaultDomain,
		},
	})
}

func (p *PocketstratorApp) handleSystemHealth(e *core.RequestEvent) error {
	validation := p.orchestrator.ValidateSystemRequirements()

	health := map[string]any{
		"healthy": validation.IsValid,
		"checks": map[string]any{
			"systemd":     len(validation.Errors) == 0,
			"caddy":       len(validation.Warnings) == 0,
			"permissions": validation.IsValid,
		},
		"errors":   validation.Errors,
		"warnings": validation.Warnings,
	}

	statusCode := 200
	if !validation.IsValid {
		statusCode = 503 // Service Unavailable
	}

	return e.JSON(statusCode, health)
}

// performHealthCheck performs health checks on all services
func (p *PocketstratorApp) performHealthCheck() {
	ctx := context.Background()

	log.Println("🔍 Performing health check on all services...")

	services, err := p.orchestrator.ListServices(ctx)
	if err != nil {
		log.Printf("❌ Failed to list services for health check: %v", err)
		return
	}

	for _, svc := range services {
		status, err := p.orchestrator.GetServiceStatus(ctx, svc.ID)
		if err != nil {
			log.Printf("❌ Failed to get status for service %s: %v", svc.ProjectName, err)
			continue
		}

		// Log status changes
		if status.IsRunning && svc.Status != "active" {
			log.Printf("✅ Service %s is now active", svc.ProjectName)
		} else if !status.IsRunning && svc.Status == "active" {
			log.Printf("⚠️  Service %s is now inactive", svc.ProjectName)
		}
	}

	log.Println("✅ Health check completed")
}

// the default pb_public dir location is relative to the executable
func defaultPublicDir() string {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		// most likely ran with go run
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}
