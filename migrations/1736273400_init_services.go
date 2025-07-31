package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	core.RegisterMigrationDown("1736273400_init_services.go", func(app core.App) error {
		// Remove the services collection
		collection, err := app.FindCollectionByNameOrId("services")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})

	core.RegisterMigrationUp("1736273400_init_services.go", func(app core.App) error {
		// Create services collection
		collection := &models.Collection{}
		collection.MarkAsNew()
		collection.Id = "pbc_1234567890" // Use a consistent ID
		collection.Name = "services"
		collection.Type = models.CollectionTypeBase
		collection.System = false

		// JSON schema definition
		jsonData := `[
			{
				"id": "text_project_name",
				"name": "project_name",
				"type": "text",
				"required": true,
				"presentable": false,
				"unique": false,
				"options": {
					"min": 1,
					"max": 50,
					"pattern": ""
				}
			},
			{
				"id": "number_port", 
				"name": "port",
				"type": "number",
				"required": true,
				"presentable": false,
				"unique": false,
				"options": {
					"min": 1024,
					"max": 65535,
					"noDecimal": true
				}
			},
			{
				"id": "text_pocketbase_version",
				"name": "pocketbase_version", 
				"type": "text",
				"required": true,
				"presentable": false,
				"unique": false,
				"options": {
					"min": 1,
					"max": 20,
					"pattern": ""
				}
			},
			{
				"id": "text_domain",
				"name": "domain",
				"type": "text", 
				"required": true,
				"presentable": false,
				"unique": false,
				"options": {
					"min": 3,
					"max": 253,
					"pattern": ""
				}
			},
			{
				"id": "select_status",
				"name": "status",
				"type": "select",
				"required": true,
				"presentable": false,
				"unique": false,
				"options": {
					"maxSelect": 1,
					"values": ["active", "inactive", "error", "deploying"]
				}
			},
			{
				"id": "text_systemd_config_hash",
				"name": "systemd_config_hash",
				"type": "text",
				"required": false,
				"presentable": false,
				"unique": false,
				"options": {
					"min": 0,
					"max": 64,
					"pattern": ""
				}
			},
			{
				"id": "text_caddy_config_hash", 
				"name": "caddy_config_hash",
				"type": "text",
				"required": false,
				"presentable": false,
				"unique": false,
				"options": {
					"min": 0,
					"max": 64,
					"pattern": ""
				}
			},
			{
				"id": "date_last_health_check",
				"name": "last_health_check",
				"type": "date",
				"required": false,
				"presentable": false,
				"unique": false,
				"options": {}
			},
			{
				"id": "text_created_by",
				"name": "created_by", 
				"type": "text",
				"required": false,
				"presentable": false,
				"unique": false,
				"options": {
					"min": 0,
					"max": 255,
					"pattern": ""
				}
			},
			{
				"id": "text_description",
				"name": "description",
				"type": "text",
				"required": false,
				"presentable": false,
				"unique": false,
				"options": {
					"min": 0,
					"max": 500,
					"pattern": ""
				}
			}
		]`

		fields := []*models.CollectionField{}
		if err := json.Unmarshal([]byte(jsonData), &fields); err != nil {
			return err
		}

		collection.Fields = fields

		// Set access rules (require authentication)
		collection.ListRule = types.Pointer("@request.auth.id != ''")
		collection.ViewRule = types.Pointer("@request.auth.id != ''")
		collection.CreateRule = types.Pointer("@request.auth.id != ''")
		collection.UpdateRule = types.Pointer("@request.auth.id != ''")
		collection.DeleteRule = types.Pointer("@request.auth.id != ''")

		return app.Save(collection)
	})
}
