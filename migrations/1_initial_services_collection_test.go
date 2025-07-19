package migrations

import (
	"os"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestServicesCollectionMigration(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "pockestrator_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PocketBase app
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: tempDir,
	})

	// Initialize the app
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap app: %v", err)
	}

	// Run migrations
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify the collection was created
	collection, err := app.FindCollectionByNameOrId("services")
	if err != nil {
		t.Fatalf("Services collection not found: %v", err)
	}

	// Verify collection properties
	if collection.Name != "services" {
		t.Errorf("Expected collection name 'services', got '%s'", collection.Name)
	}

	if collection.Type != "base" {
		t.Errorf("Expected collection type 'base', got '%s'", collection.Type)
	}

	// Verify we have the expected number of fields (5 custom fields + system fields)
	if len(collection.Fields) < 5 {
		t.Errorf("Expected at least 5 fields in collection, got %d", len(collection.Fields))
	}

	// Verify indexes exist
	if len(collection.Indexes) != 2 {
		t.Errorf("Expected 2 indexes, got %d", len(collection.Indexes))
	}

	// Check for unique indexes on name and port
	hasNameIndex := false
	hasPortIndex := false
	for _, index := range collection.Indexes {
		if index == "CREATE UNIQUE INDEX idx_services_name ON services (name)" {
			hasNameIndex = true
		}
		if index == "CREATE UNIQUE INDEX idx_services_port ON services (port)" {
			hasPortIndex = true
		}
	}

	if !hasNameIndex {
		t.Error("Missing unique index on name field")
	}

	if !hasPortIndex {
		t.Error("Missing unique index on port field")
	}
}

func TestServicesCollectionValidation(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "pockestrator_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PocketBase app
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: tempDir,
	})

	// Initialize the app
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap app: %v", err)
	}

	// Run migrations
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	collection, err := app.FindCollectionByNameOrId("services")
	if err != nil {
		t.Fatalf("Services collection not found: %v", err)
	}

	// Test valid record creation
	record := core.NewRecord(collection)
	record.Set("name", "testservice")
	record.Set("port", 8080)
	record.Set("version", "1.2.3")
	record.Set("subdomain", "testservice")
	record.Set("status", "running")

	if err := app.Save(record); err != nil {
		t.Errorf("Failed to save valid record: %v", err)
	}

	// Test duplicate name constraint
	duplicateRecord := core.NewRecord(collection)
	duplicateRecord.Set("name", "testservice") // Same name
	duplicateRecord.Set("port", 8081)          // Different port
	duplicateRecord.Set("version", "1.2.3")
	duplicateRecord.Set("subdomain", "testservice2")
	duplicateRecord.Set("status", "running")

	if err := app.Save(duplicateRecord); err == nil {
		t.Error("Expected error when saving record with duplicate name, but got none")
	}

	// Test duplicate port constraint
	duplicatePortRecord := core.NewRecord(collection)
	duplicatePortRecord.Set("name", "testservice2") // Different name
	duplicatePortRecord.Set("port", 8080)           // Same port
	duplicatePortRecord.Set("version", "1.2.3")
	duplicatePortRecord.Set("subdomain", "testservice2")
	duplicatePortRecord.Set("status", "running")

	if err := app.Save(duplicatePortRecord); err == nil {
		t.Error("Expected error when saving record with duplicate port, but got none")
	}

	// Test invalid port range
	invalidPortRecord := core.NewRecord(collection)
	invalidPortRecord.Set("name", "invalidport")
	invalidPortRecord.Set("port", 7999) // Below minimum
	invalidPortRecord.Set("version", "1.2.3")
	invalidPortRecord.Set("subdomain", "invalidport")
	invalidPortRecord.Set("status", "running")

	if err := app.Save(invalidPortRecord); err == nil {
		t.Error("Expected error when saving record with invalid port, but got none")
	}

	// Test invalid version format
	invalidVersionRecord := core.NewRecord(collection)
	invalidVersionRecord.Set("name", "invalidversion")
	invalidVersionRecord.Set("port", 8082)
	invalidVersionRecord.Set("version", "1.2") // Invalid format
	invalidVersionRecord.Set("subdomain", "invalidversion")
	invalidVersionRecord.Set("status", "running")

	if err := app.Save(invalidVersionRecord); err == nil {
		t.Error("Expected error when saving record with invalid version format, but got none")
	}

	// Test invalid status
	invalidStatusRecord := core.NewRecord(collection)
	invalidStatusRecord.Set("name", "invalidstatus")
	invalidStatusRecord.Set("port", 8083)
	invalidStatusRecord.Set("version", "1.2.3")
	invalidStatusRecord.Set("subdomain", "invalidstatus")
	invalidStatusRecord.Set("status", "invalid") // Invalid status

	if err := app.Save(invalidStatusRecord); err == nil {
		t.Error("Expected error when saving record with invalid status, but got none")
	}
}
