package services

import (
	"archive/zip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockHTTPClient implements HTTPClient for testing
type MockHTTPClient struct {
	GetFunc func(url string) (*http.Response, error)
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	if m.GetFunc != nil {
		return m.GetFunc(url)
	}
	return nil, fmt.Errorf("mock not implemented")
}

func TestNewPocketBaseManager(t *testing.T) {
	baseDir := "/tmp/test"
	manager := NewPocketBaseManager(baseDir)

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}

	if manager.baseDir != baseDir {
		t.Errorf("Expected baseDir to be %s, got %s", baseDir, manager.baseDir)
	}

	if manager.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestDownloadPocketBase(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pocketbase_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "releases/latest") {
			// Mock GitHub API response for latest version
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v0.20.0"}`)
			return
		}

		if strings.Contains(r.URL.Path, "pocketbase_0.20.0_linux_amd64.zip") {
			// Create a mock zip file with pocketbase binary
			w.Header().Set("Content-Type", "application/zip")

			// Create zip content in memory
			zipContent := createMockZip(t)
			w.Write(zipContent)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	// Create mock HTTP client
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			if strings.Contains(url, "api.github.com") {
				url = server.URL + "/releases/latest"
			} else if strings.Contains(url, "github.com/pocketbase/pocketbase/releases") {
				url = server.URL + "/pocketbase_0.20.0_linux_amd64.zip"
			}
			return http.Get(url)
		},
	}

	manager := NewPocketBaseManagerWithClient(tempDir, mockClient)
	projectName := "test-project"

	t.Run("download with latest version", func(t *testing.T) {
		err := manager.DownloadPocketBase("latest", projectName)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check if project directory was created
		projectDir := filepath.Join(tempDir, projectName)
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			t.Error("Expected project directory to be created")
		}
	})

	t.Run("download with specific version", func(t *testing.T) {
		projectName2 := "test-project-2"
		err := manager.DownloadPocketBase("0.20.0", projectName2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check if project directory was created
		projectDir := filepath.Join(tempDir, projectName2)
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			t.Error("Expected project directory to be created")
		}
	})
}

func TestExtractPocketBase(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pocketbase_extract_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewPocketBaseManager(tempDir)
	projectName := "test-project"
	version := "0.20.0"

	// Create project directory and mock zip file
	projectDir := filepath.Join(tempDir, projectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	zipPath := filepath.Join(projectDir, fmt.Sprintf("pocketbase_%s.zip", version))
	zipContent := createMockZip(t)
	if err := os.WriteFile(zipPath, zipContent, 0644); err != nil {
		t.Fatalf("Failed to create mock zip file: %v", err)
	}

	t.Run("extract pocketbase binary", func(t *testing.T) {
		err := manager.ExtractPocketBase(projectName, version)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check if binary was extracted
		binaryPath := filepath.Join(projectDir, "pocketbase")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Error("Expected pocketbase binary to be extracted")
		}

		// Check if zip file was removed
		if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
			t.Error("Expected zip file to be removed after extraction")
		}
	})
}

func TestSetPermissions(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pocketbase_permissions_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewPocketBaseManager(tempDir)
	projectName := "test-project"

	// Create project directory and mock binary
	projectDir := filepath.Join(tempDir, projectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	binaryPath := filepath.Join(projectDir, "pocketbase")
	if err := os.WriteFile(binaryPath, []byte("mock binary"), 0644); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	t.Run("set executable permissions", func(t *testing.T) {
		err := manager.SetPermissions(projectName)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check if permissions were set correctly
		info, err := os.Stat(binaryPath)
		if err != nil {
			t.Fatalf("Failed to stat binary: %v", err)
		}

		mode := info.Mode()
		if mode&0111 == 0 {
			t.Error("Expected binary to have execute permissions")
		}
	})
}

func TestCreateSuperUser(t *testing.T) {
	// This test is more complex as it requires mocking exec.Command
	// For now, we'll test the basic error handling
	tempDir, err := os.MkdirTemp("", "pocketbase_superuser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewPocketBaseManager(tempDir)
	projectName := "test-project"

	t.Run("create superuser with missing binary", func(t *testing.T) {
		err := manager.CreateSuperUser(projectName, "admin@example.com", "password123")
		if err == nil {
			t.Error("Expected error when binary doesn't exist")
		}
	})
}

func TestGetLatestVersion(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases/latest" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v0.20.0"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Create mock HTTP client
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			if strings.Contains(url, "api.github.com") {
				url = server.URL + "/releases/latest"
			}
			return http.Get(url)
		},
	}

	manager := NewPocketBaseManagerWithClient("/tmp", mockClient)

	t.Run("get latest version", func(t *testing.T) {
		version, err := manager.getLatestVersion()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if version != "0.20.0" {
			t.Errorf("Expected version 0.20.0, got %s", version)
		}
	})
}

// createMockZip creates a mock zip file containing a pocketbase binary
func createMockZip(t *testing.T) []byte {
	t.Helper()

	// Create zip in memory
	var buf strings.Builder
	writer := zip.NewWriter(&buf)

	// Add pocketbase file to zip
	file, err := writer.Create("pocketbase")
	if err != nil {
		t.Fatalf("Failed to create file in zip: %v", err)
	}

	_, err = file.Write([]byte("mock pocketbase binary"))
	if err != nil {
		t.Fatalf("Failed to write to zip file: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	return []byte(buf.String())
}
