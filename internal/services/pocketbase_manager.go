package services

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HTTPClient interface for mocking HTTP requests
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// PocketBaseManagerImpl implements the PocketBaseManager interface
type PocketBaseManagerImpl struct {
	httpClient HTTPClient
	baseDir    string
}

// NewPocketBaseManager creates a new PocketBase manager instance
func NewPocketBaseManager(baseDir string) *PocketBaseManagerImpl {
	return &PocketBaseManagerImpl{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseDir: baseDir,
	}
}

// NewPocketBaseManagerWithClient creates a new PocketBase manager with custom HTTP client
func NewPocketBaseManagerWithClient(baseDir string, client HTTPClient) *PocketBaseManagerImpl {
	return &PocketBaseManagerImpl{
		httpClient: client,
		baseDir:    baseDir,
	}
}

// DownloadPocketBase downloads the specified PocketBase version
func (pm *PocketBaseManagerImpl) DownloadPocketBase(version string, projectName string) error {
	if version == "" || version == "latest" {
		var err error
		version, err = pm.getLatestVersion()
		if err != nil {
			return fmt.Errorf("failed to get latest version: %w", err)
		}
	}

	// Create project directory
	projectDir := filepath.Join(pm.baseDir, projectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Download URL for Linux AMD64
	downloadURL := fmt.Sprintf("https://github.com/pocketbase/pocketbase/releases/download/v%s/pocketbase_%s_linux_amd64.zip", version, version)

	// Download the file
	zipPath := filepath.Join(projectDir, fmt.Sprintf("pocketbase_%s.zip", version))
	if err := pm.downloadFile(downloadURL, zipPath); err != nil {
		return fmt.Errorf("failed to download PocketBase: %w", err)
	}

	return nil
}

// ExtractPocketBase extracts the downloaded PocketBase archive
func (pm *PocketBaseManagerImpl) ExtractPocketBase(projectName string, version string) error {
	projectDir := filepath.Join(pm.baseDir, projectName)
	zipPath := filepath.Join(projectDir, fmt.Sprintf("pocketbase_%s.zip", version))

	// Open the zip file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// Extract files
	for _, file := range reader.File {
		if file.Name == "pocketbase" {
			if err := pm.extractFile(file, filepath.Join(projectDir, "pocketbase")); err != nil {
				return fmt.Errorf("failed to extract pocketbase binary: %w", err)
			}
			break
		}
	}

	// Remove the zip file after extraction
	if err := os.Remove(zipPath); err != nil {
		return fmt.Errorf("failed to remove zip file: %w", err)
	}

	return nil
}

// SetPermissions sets executable permissions on the PocketBase binary
func (pm *PocketBaseManagerImpl) SetPermissions(projectName string) error {
	binaryPath := filepath.Join(pm.baseDir, projectName, "pocketbase")

	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// CreateSuperUser creates a default superuser account for the PocketBase instance
func (pm *PocketBaseManagerImpl) CreateSuperUser(projectName string, email string, password string) error {
	binaryPath := filepath.Join(pm.baseDir, projectName, "pocketbase")
	projectDir := filepath.Join(pm.baseDir, projectName)

	// Create superuser using PocketBase CLI
	cmd := exec.Command(binaryPath, "admin", "create", email, password)
	cmd.Dir = projectDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create superuser: %w, output: %s", err, string(output))
	}

	return nil
}

// getLatestVersion fetches the latest PocketBase version from GitHub API
func (pm *PocketBaseManagerImpl) getLatestVersion() (string, error) {
	resp, err := pm.httpClient.Get("https://api.github.com/repos/pocketbase/pocketbase/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Simple JSON parsing to extract tag_name
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract version from tag_name (e.g., "v0.20.0" -> "0.20.0")
	bodyStr := string(body)
	tagStart := strings.Index(bodyStr, `"tag_name":"v`)
	if tagStart == -1 {
		return "", fmt.Errorf("could not find tag_name in response")
	}

	tagStart += len(`"tag_name":"v`)
	tagEnd := strings.Index(bodyStr[tagStart:], `"`)
	if tagEnd == -1 {
		return "", fmt.Errorf("could not parse tag_name from response")
	}

	version := bodyStr[tagStart : tagStart+tagEnd]
	return version, nil
}

// downloadFile downloads a file from the given URL to the specified path
func (pm *PocketBaseManagerImpl) downloadFile(url string, filepath string) error {
	resp, err := pm.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractFile extracts a single file from a zip archive
func (pm *PocketBaseManagerImpl) extractFile(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in archive: %w", err)
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}
