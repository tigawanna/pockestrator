package caddy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

// ConfigTemplate is the Caddy configuration template for a service
const ConfigTemplate = `
{{.Subdomain}}.{{.Domain}} {
    request_body {
        max_size 10MB
    }
    reverse_proxy 127.0.0.1:{{.Port}} {
        transport http {
            read_timeout 360s
        }
        # Add these headers to forward client IP
        header_up X-Forwarded-For {remote_host}
        header_up X-Real-IP {remote_host}
    }
}
`

// Manager handles Caddy configuration operations
type Manager struct {
	caddyfilePath string
}

// ServiceConfig holds the configuration for generating Caddy config
type ServiceConfig struct {
	Subdomain string
	Domain    string
	Port      int
}

// NewManager creates a new Caddy manager
func NewManager(caddyfilePath string) *Manager {
	return &Manager{
		caddyfilePath: caddyfilePath,
	}
}

// AddService adds a new service configuration to Caddyfile
func (m *Manager) AddService(config *ServiceConfig) error {
	// Parse template
	tmpl, err := template.New("caddy").Parse(ConfigTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse Caddy template: %w", err)
	}

	// Generate config string
	var configStr strings.Builder
	if err := tmpl.Execute(&configStr, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Check if Caddyfile exists
	if _, err := os.Stat(m.caddyfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Caddyfile not found at %s", m.caddyfilePath)
	}

	// Check if configuration already exists
	exists, err := m.configExists(config.Subdomain, config.Domain)
	if err != nil {
		return fmt.Errorf("failed to check existing config: %w", err)
	}
	if exists {
		return fmt.Errorf("configuration for %s.%s already exists", config.Subdomain, config.Domain)
	}

	// Append to Caddyfile
	file, err := os.OpenFile(m.caddyfilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open Caddyfile: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(configStr.String()); err != nil {
		return fmt.Errorf("failed to write to Caddyfile: %w", err)
	}

	return nil
}

// RemoveService removes a service configuration from Caddyfile
func (m *Manager) RemoveService(subdomain, domain string) error {
	// Read the entire file
	content, err := os.ReadFile(m.caddyfilePath)
	if err != nil {
		return fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	// Create regex pattern to match the service block
	pattern := fmt.Sprintf(`(?s)%s\.%s\s*\{[^}]*\}`, regexp.QuoteMeta(subdomain), regexp.QuoteMeta(domain))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to compile regex: %w", err)
	}

	// Remove the configuration block
	newContent := re.ReplaceAll(content, []byte(""))

	// Write back to file
	if err := os.WriteFile(m.caddyfilePath, newContent, 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	return nil
}

// ValidateConfig validates the Caddy configuration
func (m *Manager) ValidateConfig() error {
	cmd := exec.Command("caddy", "validate", "--config", m.caddyfilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Caddy config validation failed: %s", string(output))
	}
	return nil
}

// ReloadConfig reloads the Caddy configuration
func (m *Manager) ReloadConfig() error {
	// First validate
	if err := m.ValidateConfig(); err != nil {
		return fmt.Errorf("config validation failed before reload: %w", err)
	}

	// Reload Caddy
	cmd := exec.Command("sudo", "systemctl", "reload", "caddy")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	return nil
}

// GetServiceConfig extracts the configuration for a specific service
func (m *Manager) GetServiceConfig(subdomain, domain string) (string, error) {
	content, err := os.ReadFile(m.caddyfilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	// Create regex pattern to match the service block
	pattern := fmt.Sprintf(`(?s)(%s\.%s\s*\{[^}]*\})`, regexp.QuoteMeta(subdomain), regexp.QuoteMeta(domain))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	matches := re.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		return "", fmt.Errorf("configuration not found for %s.%s", subdomain, domain)
	}

	return matches[1], nil
}

// UpdateServiceConfig updates an existing service configuration
func (m *Manager) UpdateServiceConfig(oldSubdomain, oldDomain string, newConfig *ServiceConfig) error {
	// Remove old configuration
	if err := m.RemoveService(oldSubdomain, oldDomain); err != nil {
		return fmt.Errorf("failed to remove old config: %w", err)
	}

	// Add new configuration
	if err := m.AddService(newConfig); err != nil {
		return fmt.Errorf("failed to add new config: %w", err)
	}

	return nil
}

// ListServices returns a list of all configured services
func (m *Manager) ListServices() ([]string, error) {
	content, err := os.ReadFile(m.caddyfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	// Regex to find all site blocks
	re := regexp.MustCompile(`([a-zA-Z0-9\-\.]+)\s*\{`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	var services []string
	for _, match := range matches {
		if len(match) > 1 {
			services = append(services, match[1])
		}
	}

	return services, nil
}

// configExists checks if a configuration block already exists
func (m *Manager) configExists(subdomain, domain string) (bool, error) {
	file, err := os.Open(m.caddyfilePath)
	if err != nil {
		return false, fmt.Errorf("failed to open Caddyfile: %w", err)
	}
	defer file.Close()

	pattern := fmt.Sprintf("%s.%s", subdomain, domain)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if strings.Contains(scanner.Text(), pattern) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to scan Caddyfile: %w", err)
	}

	return false, nil
}

// IsCaddyRunning checks if Caddy service is running
func (m *Manager) IsCaddyRunning() (bool, error) {
	cmd := exec.Command("sudo", "systemctl", "is-active", "caddy")
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	status := strings.TrimSpace(string(output))
	return status == "active", nil
}

// GetCaddyStatus returns the status of the Caddy service
func (m *Manager) GetCaddyStatus() (string, error) {
	cmd := exec.Command("sudo", "systemctl", "status", "caddy", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Caddy status: %w", err)
	}

	return string(output), nil
}

// BackupConfig creates a backup of the current Caddyfile
func (m *Manager) BackupConfig() (string, error) {
	backupPath := m.caddyfilePath + ".backup"

	content, err := os.ReadFile(m.caddyfilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// RestoreConfig restores the Caddyfile from a backup
func (m *Manager) RestoreConfig(backupPath string) error {
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(m.caddyfilePath, content, 0644); err != nil {
		return fmt.Errorf("failed to restore Caddyfile: %w", err)
	}

	return nil
}
