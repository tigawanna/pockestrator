package database

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ServiceRecord represents a service record in the database
type ServiceRecord struct {
	ID                string    `json:"id" db:"id"`
	ProjectName       string    `json:"project_name" db:"project_name"`
	Port              int       `json:"port" db:"port"`
	PocketBaseVersion string    `json:"pocketbase_version" db:"pocketbase_version"`
	Domain            string    `json:"domain" db:"domain"`
	Status            string    `json:"status" db:"status"`
	SystemdConfigHash string    `json:"systemd_config_hash" db:"systemd_config_hash"`
	CaddyConfigHash   string    `json:"caddy_config_hash" db:"caddy_config_hash"`
	LastHealthCheck   time.Time `json:"last_health_check" db:"last_health_check"`
	CreatedBy         string    `json:"created_by" db:"created_by"`
	CreatedAt         time.Time `json:"created" db:"created"`
	UpdatedAt         time.Time `json:"updated" db:"updated"`
}

// Manager handles database operations
type Manager struct {
	app *pocketbase.PocketBase
}

// NewManager creates a new database manager
func NewManager(app *pocketbase.PocketBase) *Manager {
	return &Manager{app: app}
}

// CreateService creates a new service record
func (m *Manager) CreateService(ctx context.Context, service *ServiceRecord) error {
	collection, err := m.app.FindCollectionByNameOrId("services")
	if err != nil {
		return fmt.Errorf("failed to find services collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("project_name", service.ProjectName)
	record.Set("port", service.Port)
	record.Set("pocketbase_version", service.PocketBaseVersion)
	record.Set("domain", service.Domain)
	record.Set("status", service.Status)
	record.Set("systemd_config_hash", service.SystemdConfigHash)
	record.Set("caddy_config_hash", service.CaddyConfigHash)
	record.Set("last_health_check", service.LastHealthCheck)
	record.Set("created_by", service.CreatedBy)

	if err := m.app.Save(record); err != nil {
		return fmt.Errorf("failed to create service record: %w", err)
	}

	service.ID = record.Id
	service.CreatedAt = record.GetDateTime("created").Time()
	service.UpdatedAt = record.GetDateTime("updated").Time()

	return nil
}

// GetService retrieves a service by ID
func (m *Manager) GetService(ctx context.Context, id string) (*ServiceRecord, error) {
	record, err := m.app.FindRecordById("services", id)
	if err != nil {
		return nil, fmt.Errorf("failed to find service: %w", err)
	}

	return m.recordToService(record), nil
}

// GetServiceByName retrieves a service by project name
func (m *Manager) GetServiceByName(ctx context.Context, projectName string) (*ServiceRecord, error) {
	record, err := m.app.FindFirstRecordByFilter("services", "project_name = {:name}", map[string]any{
		"name": projectName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find service by name: %w", err)
	}

	return m.recordToService(record), nil
}

// ListServices retrieves all services
func (m *Manager) ListServices(ctx context.Context) ([]*ServiceRecord, error) {
	records, err := m.app.FindRecordsByFilter("services", "", "", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	services := make([]*ServiceRecord, len(records))
	for i, record := range records {
		services[i] = m.recordToService(record)
	}

	return services, nil
}

// UpdateService updates an existing service record
func (m *Manager) UpdateService(ctx context.Context, service *ServiceRecord) error {
	record, err := m.app.FindRecordById("services", service.ID)
	if err != nil {
		return fmt.Errorf("failed to find service record: %w", err)
	}

	record.Set("project_name", service.ProjectName)
	record.Set("port", service.Port)
	record.Set("pocketbase_version", service.PocketBaseVersion)
	record.Set("domain", service.Domain)
	record.Set("status", service.Status)
	record.Set("systemd_config_hash", service.SystemdConfigHash)
	record.Set("caddy_config_hash", service.CaddyConfigHash)
	record.Set("last_health_check", service.LastHealthCheck)

	if err := m.app.Save(record); err != nil {
		return fmt.Errorf("failed to update service record: %w", err)
	}

	service.UpdatedAt = record.GetDateTime("updated").Time()

	return nil
}

// DeleteService deletes a service record
func (m *Manager) DeleteService(ctx context.Context, id string) error {
	record, err := m.app.FindRecordById("services", id)
	if err != nil {
		return fmt.Errorf("failed to find service record: %w", err)
	}

	if err := m.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete service record: %w", err)
	}

	return nil
}

// GetUsedPorts returns a list of all used ports
func (m *Manager) GetUsedPorts(ctx context.Context) ([]int, error) {
	records, err := m.app.FindRecordsByFilter("services", "", "", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get used ports: %w", err)
	}

	ports := make([]int, len(records))
	for i, record := range records {
		ports[i] = record.GetInt("port")
	}

	return ports, nil
}

// GetExistingServices returns a list of all existing service names
func (m *Manager) GetExistingServices(ctx context.Context) ([]string, error) {
	records, err := m.app.FindRecordsByFilter("services", "", "", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing services: %w", err)
	}

	services := make([]string, len(records))
	for i, record := range records {
		services[i] = record.GetString("project_name")
	}

	return services, nil
}

// UpdateServiceStatus updates the status of a service
func (m *Manager) UpdateServiceStatus(ctx context.Context, id, status string) error {
	record, err := m.app.FindRecordById("services", id)
	if err != nil {
		return fmt.Errorf("failed to find service record: %w", err)
	}

	record.Set("status", status)
	record.Set("last_health_check", time.Now())

	if err := m.app.Save(record); err != nil {
		return fmt.Errorf("failed to update service status: %w", err)
	}

	return nil
}

// UpdateConfigHashes updates the configuration hashes for a service
func (m *Manager) UpdateConfigHashes(ctx context.Context, id, systemdHash, caddyHash string) error {
	record, err := m.app.FindRecordById("services", id)
	if err != nil {
		return fmt.Errorf("failed to find service record: %w", err)
	}

	record.Set("systemd_config_hash", systemdHash)
	record.Set("caddy_config_hash", caddyHash)

	if err := m.app.Save(record); err != nil {
		return fmt.Errorf("failed to update config hashes: %w", err)
	}

	return nil
}

// recordToService converts a PocketBase record to a ServiceRecord
func (m *Manager) recordToService(record *core.Record) *ServiceRecord {
	return &ServiceRecord{
		ID:                record.Id,
		ProjectName:       record.GetString("project_name"),
		Port:              record.GetInt("port"),
		PocketBaseVersion: record.GetString("pocketbase_version"),
		Domain:            record.GetString("domain"),
		Status:            record.GetString("status"),
		SystemdConfigHash: record.GetString("systemd_config_hash"),
		CaddyConfigHash:   record.GetString("caddy_config_hash"),
		LastHealthCheck:   record.GetDateTime("last_health_check").Time(),
		CreatedBy:         record.GetString("created_by"),
		CreatedAt:         record.GetDateTime("created").Time(),
		UpdatedAt:         record.GetDateTime("updated").Time(),
	}
}

// GenerateConfigHash generates a hash for configuration content
func GenerateConfigHash(content string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(content)))
}

// IsServiceHealthy checks if a service's configurations are consistent
func (m *Manager) IsServiceHealthy(ctx context.Context, service *ServiceRecord, systemdConfig, caddyConfig string) bool {
	systemdHash := GenerateConfigHash(systemdConfig)
	caddyHash := GenerateConfigHash(caddyConfig)

	return service.SystemdConfigHash == systemdHash && service.CaddyConfigHash == caddyHash
}
