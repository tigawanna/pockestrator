package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tigawanna/pockestrator/internal/models"
)

// CaddyManagerImpl implements the CaddyManager interface
type CaddyManagerImpl struct {
	caddyfilePath string
	domain        string
}

// NewCaddyManager creates a new Caddy manager instance with default configuration
func NewCaddyManager(domain string) *CaddyManagerImpl {
	return &CaddyManagerImpl{
		caddyfilePath: "/etc/caddy/Caddyfile",
		domain:        domain,
	}
}

// NewCaddyManagerWithPath creates a new Caddy manager with custom Caddyfile path
func NewCaddyManagerWithPath(caddyfilePath string, domain string) *CaddyManagerImpl {
	return &CaddyManagerImpl{
		caddyfilePath: caddyfilePath,
		domain:        domain,
	}
}

// AddConfiguration adds a reverse proxy configuration for the given service to the Caddyfile
func (cm *CaddyManagerImpl) AddConfiguration(service *models.Service) error {
	// Read current Caddyfile content
	content, err := cm.readCaddyfile()
	if err != nil {
		return fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	// Check if configuration already exists
	if cm.configExists(content, service.Subdomain) {
		// Remove existing configuration first
		content = cm.removeConfig(content, service.Subdomain)
	}

	// Generate new configuration block
	newConfig := cm.generateConfig(service)

	// Append new configuration
	updatedContent := content
	if !strings.HasSuffix(updatedContent, "\n") {
		updatedContent += "\n"
	}
	updatedContent += newConfig

	// Write updated Caddyfile
	if err := cm.writeCaddyfile(updatedContent); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	// Reload Caddy to apply changes
	return cm.ReloadCaddy()
}

// RemoveConfiguration removes the configuration for the given service from the Caddyfile
func (cm *CaddyManagerImpl) RemoveConfiguration(serviceName string) error {
	// Read current Caddyfile content
	content, err := cm.readCaddyfile()
	if err != nil {
		return fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	// Generate subdomain from service name for consistency
	subdomain := serviceName

	// Remove configuration if it exists
	if cm.configExists(content, subdomain) {
		updatedContent := cm.removeConfig(content, subdomain)

		// Write updated Caddyfile
		if err := cm.writeCaddyfile(updatedContent); err != nil {
			return fmt.Errorf("failed to write Caddyfile: %w", err)
		}

		// Reload Caddy to apply changes
		return cm.ReloadCaddy()
	}

	// Configuration didn't exist, no changes needed
	return nil
}

// ReloadCaddy reloads the Caddy server to apply configuration changes
func (cm *CaddyManagerImpl) ReloadCaddy() error {
	cmd := exec.Command("systemctl", "reload", "caddy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload Caddy: %w, output: %s", err, string(output))
	}
	return nil
}

// ValidateConfiguration checks if the Caddy configuration for the given service is correct
func (cm *CaddyManagerImpl) ValidateConfiguration(service *models.Service) bool {
	// Read current Caddyfile content
	content, err := cm.readCaddyfile()
	if err != nil {
		return false
	}

	// Check if configuration exists
	if !cm.configExists(content, service.Subdomain) {
		return false
	}

	sanitizedContent := strings.ReplaceAll(content, "\r\n", "\n")

	// Extract the service's configuration block
	pattern := fmt.Sprintf(`%s\.%s \{[^}]*\}`, regexp.QuoteMeta(service.Subdomain), regexp.QuoteMeta(cm.domain))
	re := regexp.MustCompile(pattern)
	match := re.FindString(sanitizedContent)

	if match == "" {
		return false
	}

	// Check if the port in the configuration matches the service port
	portPattern := fmt.Sprintf(`reverse_proxy 127\.0\.0\.1:%d`, service.Port)
	portRe := regexp.MustCompile(portPattern)

	return portRe.MatchString(match)
}

// readCaddyfile reads the content of the Caddyfile
func (cm *CaddyManagerImpl) readCaddyfile() (string, error) {
	// Create directory and empty file if it doesn't exist
	dir := filepath.Dir(cm.caddyfilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory for Caddyfile: %w", err)
	}

	// Check if file exists, create it if not
	if _, err := os.Stat(cm.caddyfilePath); os.IsNotExist(err) {
		if err := os.WriteFile(cm.caddyfilePath, []byte(""), 0644); err != nil {
			return "", fmt.Errorf("failed to create Caddyfile: %w", err)
		}
	}

	// Read file content
	content, err := os.ReadFile(cm.caddyfilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	return string(content), nil
}

// writeCaddyfile writes content to the Caddyfile
func (cm *CaddyManagerImpl) writeCaddyfile(content string) error {
	return os.WriteFile(cm.caddyfilePath, []byte(content), 0644)
}

// configExists checks if a configuration for the given subdomain already exists
func (cm *CaddyManagerImpl) configExists(content string, subdomain string) bool {
	pattern := fmt.Sprintf(`%s\.%s \{`, regexp.QuoteMeta(subdomain), regexp.QuoteMeta(cm.domain))
	return regexp.MustCompile(pattern).MatchString(content)
}

// removeConfig removes the configuration block for the given subdomain
func (cm *CaddyManagerImpl) removeConfig(content string, subdomain string) string {
	// Define pattern to match the entire configuration block
	pattern := fmt.Sprintf(`%s\.%s \{[^}]*\}\s*`, regexp.QuoteMeta(subdomain), regexp.QuoteMeta(cm.domain))
	re := regexp.MustCompile(pattern)

	// Remove the configuration block
	return re.ReplaceAllString(content, "")
}

// generateConfig generates a Caddy configuration block for the given service
func (cm *CaddyManagerImpl) generateConfig(service *models.Service) string {
	return fmt.Sprintf(`%s.%s {
    reverse_proxy 127.0.0.1:%d
    header Strict-Transport-Security max-age=31536000;
    encode gzip
}

`, service.Subdomain, cm.domain, service.Port)
}
