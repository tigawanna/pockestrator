package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("services")

		// Add name field
		nameField := &core.TextField{}
		nameField.Name = "name"
		nameField.Required = true
		nameField.Presentable = true
		nameField.Min = 1
		nameField.Max = 50
		nameField.Pattern = `^[a-zA-Z0-9_-]+$`
		collection.Fields.Add(nameField)

		// Add port field
		portField := &core.NumberField{}
		portField.Name = "port"
		portField.Required = true
		portField.Min = types.Pointer(8000.0)
		portField.Max = types.Pointer(9999.0)
		collection.Fields.Add(portField)

		// Add version field
		versionField := &core.TextField{}
		versionField.Name = "version"
		versionField.Required = true
		versionField.Min = 1
		versionField.Max = 20
		versionField.Pattern = `^\d+\.\d+\.\d+$`
		collection.Fields.Add(versionField)

		// Add subdomain field
		subdomainField := &core.TextField{}
		subdomainField.Name = "subdomain"
		subdomainField.Required = true
		subdomainField.Min = 1
		subdomainField.Max = 63
		subdomainField.Pattern = `^[a-zA-Z0-9-]+$`
		collection.Fields.Add(subdomainField)

		// Add status field
		statusField := &core.SelectField{}
		statusField.Name = "status"
		statusField.Required = true
		statusField.MaxSelect = 1
		statusField.Values = []string{"creating", "running", "stopped", "error"}
		collection.Fields.Add(statusField)

		// Add unique indexes
		collection.Indexes = []string{
			"CREATE UNIQUE INDEX idx_services_name ON services (name)",
			"CREATE UNIQUE INDEX idx_services_port ON services (port)",
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: delete the services collection
		collection, err := app.FindCollectionByNameOrId("services")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
