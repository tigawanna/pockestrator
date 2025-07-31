package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/tigawanna/pockestrator/internal/handlers"
	"github.com/tigawanna/pockestrator/internal/hooks"
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
	"github.com/tigawanna/pockestrator/internal/validators"
)

//go:embed pb_public/dist
var distDir embed.FS

func main() {
	app := pocketbase.New()

	// ---------------------------------------------------------------
	// Configuration flags:
	// ---------------------------------------------------------------

	var migrationsDir string
	app.RootCmd.PersistentFlags().StringVar(
		&migrationsDir,
		"migrationsDir",
		"",
		"the directory with the user defined migrations",
	)

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(
		&automigrate,
		"automigrate",
		true,
		"enable/disable auto migrations",
	)

	var publicDir string
	app.RootCmd.PersistentFlags().StringVar(
		&publicDir,
		"publicDir",
		defaultPublicDir(),
		"the directory to serve static files",
	)

	var indexFallback bool
	app.RootCmd.PersistentFlags().BoolVar(
		&indexFallback,
		"indexFallback",
		true,
		"fallback the request to index.html on missing static path, e.g. when pretty urls are used with SPA",
	)

	app.RootCmd.ParseFlags(os.Args[1:])

	// ---------------------------------------------------------------
	// Plugins and configuration:
	// ---------------------------------------------------------------

	// migrate command
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo,
		Automigrate:  automigrate,
		Dir:          migrationsDir,
	})

	// static route to serve files from the provided public dir or embedded dist directory
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				// Check if we're running in dev mode with a local public directory
				if _, err := os.Stat(publicDir); err == nil {
					// Use the local public directory in development
					e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), indexFallback))
				} else {
					// Use the embedded files in production
					subFS, err := fs.Sub(distDir, "pb_public/dist")
					if err != nil {
						return err
					}
					e.Router.GET("/{path...}", apis.Static(subFS, indexFallback))
				}
			}

			return e.Next()
		},
		Priority: 999, // execute as latest as possible to allow users to provide their own route
	})

	// Initialize system managers
	systemdManager := services.NewSystemdManager()
	caddyManager := services.NewCaddyManager("example.com") // Replace with your actual domain

	// Initialize service validator with proper dependencies
	serviceValidator := validators.NewServiceValidator(app.Dao(), systemdManager, caddyManager, "/home/ubuntu")

	// Initialize service handler
	serviceHandler := handlers.NewServiceHandler(serviceValidator)

	// Register custom API routes
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Service validation endpoints
		e.Router.AddRoute(apis.Route{
			Method:  http.MethodGet,
			Path:    "/api/services/:id/validate",
			Handler: serviceHandler.ValidateService,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		// Service logs endpoint
		e.Router.AddRoute(apis.Route{
			Method:  http.MethodGet,
			Path:    "/api/services/:id/logs",
			Handler: serviceHandler.GetServiceLogs,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		// Service restart endpoint
		e.Router.AddRoute(apis.Route{
			Method:  http.MethodPost,
			Path:    "/api/services/:id/restart",
			Handler: serviceHandler.RestartService,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		// Available ports endpoint
		e.Router.AddRoute(apis.Route{
			Method:  http.MethodGet,
			Path:    "/api/system/ports/available",
			Handler: serviceHandler.GetAvailablePorts,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		// File management endpoints
		e.Router.AddRoute(apis.Route{
			Method:  http.MethodPost,
			Path:    "/api/services/:id/upload",
			Handler: serviceHandler.UploadServiceFile,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		e.Router.AddRoute(apis.Route{
			Method:  http.MethodGet,
			Path:    "/api/services/:id/files",
			Handler: serviceHandler.ListServiceFiles,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		e.Router.AddRoute(apis.Route{
			Method:  http.MethodDelete,
			Path:    "/api/services/:id/files/:filename",
			Handler: serviceHandler.DeleteServiceFile,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		// Config sync endpoint
		e.Router.AddRoute(apis.Route{
			Method:  http.MethodPost,
			Path:    "/api/services/:id/sync-config",
			Handler: serviceHandler.SyncServiceConfig,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		// PocketBase version management endpoints
		e.Router.AddRoute(apis.Route{
			Method:  http.MethodGet,
			Path:    "/api/pocketbase/versions",
			Handler: serviceHandler.ListPocketBaseVersions,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		e.Router.AddRoute(apis.Route{
			Method:  http.MethodPost,
			Path:    "/api/services/:id/update-pocketbase",
			Handler: serviceHandler.UpdatePocketBase,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		return nil
	})

	// Initialize and register service hooks
	baseDir := "/home/ubuntu"
	domain := "example.com"               // Replace with your actual domain
	defaultEmail := "admin@example.com"   // Default admin email for new services
	defaultPassword := "adminpassword123" // Default admin password for new services

	// Initialize service managers
	pbManager := services.NewPocketBaseManager()

	// Create a repository adapter for the PocketBase DAO
	repo := &PocketBaseServiceRepository{dao: app.Dao()}

	// Initialize config sync service
	configSync := services.NewConfigSyncService(
		systemdManager,
		caddyManager,
		baseDir,
		"/lib/systemd/system",
		"/etc/caddy/Caddyfile",
		domain,
		repo,
	)

	// Initialize service hooks
	serviceHooks := hooks.NewServiceHooks(
		configSync,
		pbManager,
		systemdManager,
		caddyManager,
		serviceValidator,
		baseDir,
		defaultEmail,
		defaultPassword,
	)

	// Initialize logger
	logger, err := services.NewLogger(services.LogLevelInfo, "./logs")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Create service logger factory
	loggerFactory := services.NewServiceLoggerFactory(logger, "./logs/services")

	// Update service hooks with logger
	serviceHooks.SetLogger(logger)

	// Register service hooks
	if err := serviceHooks.RegisterHooks(app); err != nil {
		logger.Fatal("Failed to register service hooks: %v", err)
	}

	// Initialize and register config sync hooks
	configSyncHooks := hooks.NewConfigSyncHooks(configSync)
	configSyncHooks.SetLogger(logger)
	configSyncHooks.RegisterConfigSyncEndpoints(app)

	// Initialize and register response middleware
	responseMiddleware := hooks.NewResponseMiddleware(logger)
	if err := responseMiddleware.Register(app); err != nil {
		logger.Fatal("Failed to register response middleware: %v", err)
	}

	log.Println("Service hooks and middleware registered successfully")

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// the default pb_public dir location is relative to the executable
func defaultPublicDir() string {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		// most likely ran with go run
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}

// PocketBaseServiceRepository is an adapter that implements the services.ServiceRepository interface
// using PocketBase's DAO
type PocketBaseServiceRepository struct {
	dao *core.Dao
}

// FindServiceByName finds a service by name
func (r *PocketBaseServiceRepository) FindServiceByName(name string) (*models.Service, error) {
	record, err := r.dao.FindFirstRecordByData("services", "name", name)
	if err != nil {
		return nil, err
	}
	return &models.Service{
		ID:        record.Id,
		Name:      record.GetString("name"),
		Port:      int(record.GetInt("port")),
		Version:   record.GetString("version"),
		Subdomain: record.GetString("subdomain"),
		Status:    record.GetString("status"),
		CreatedAt: record.GetString("created"),
		UpdatedAt: record.GetString("updated"),
	}, nil
}

// FindServiceByPort finds a service by port
func (r *PocketBaseServiceRepository) FindServiceByPort(port int) (*models.Service, error) {
	record, err := r.dao.FindFirstRecordByData("services", "port", port)
	if err != nil {
		return nil, err
	}
	return &models.Service{
		ID:        record.Id,
		Name:      record.GetString("name"),
		Port:      int(record.GetInt("port")),
		Version:   record.GetString("version"),
		Subdomain: record.GetString("subdomain"),
		Status:    record.GetString("status"),
		CreatedAt: record.GetString("created"),
		UpdatedAt: record.GetString("updated"),
	}, nil
}

// ListAllServices lists all services
func (r *PocketBaseServiceRepository) ListAllServices() ([]*models.Service, error) {
	records, err := r.dao.FindRecordsByExpr("services")
	if err != nil {
		return nil, err
	}

	services := make([]*models.Service, 0, len(records))
	for _, record := range records {
		services = append(services, &models.Service{
			ID:        record.Id,
			Name:      record.GetString("name"),
			Port:      int(record.GetInt("port")),
			Version:   record.GetString("version"),
			Subdomain: record.GetString("subdomain"),
			Status:    record.GetString("status"),
			CreatedAt: record.GetString("created"),
			UpdatedAt: record.GetString("updated"),
		})
	}

	return services, nil
}

// UpdateService updates a service
func (r *PocketBaseServiceRepository) UpdateService(service *models.Service) error {
	record, err := r.dao.FindRecordById("services", service.ID)
	if err != nil {
		return err
	}

	record.Set("name", service.Name)
	record.Set("port", service.Port)
	record.Set("version", service.Version)
	record.Set("subdomain", service.Subdomain)
	record.Set("status", service.Status)

	return r.dao.SaveRecord(record)
}
