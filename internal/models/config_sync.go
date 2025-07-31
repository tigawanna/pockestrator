package models

// ConfigItem represents a configuration item that can be synced between the database and system files
type ConfigItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"` // systemd, caddy, binary
	CollectionValue string `json:"collectionValue"`
	FileValue       string `json:"fileValue"`
	Status          string `json:"status"` // synced, conflict, missing_file, missing_collection
}

// ConfigSyncResponse represents the response for a configuration sync status request
type ConfigSyncResponse struct {
	Items     []ConfigItem `json:"items"`
	Timestamp string       `json:"timestamp"`
}

// ConfigConflict represents a conflict between a service record and system files
type ConfigConflict struct {
	ServiceID      string            `json:"serviceId"`
	ServiceName    string            `json:"serviceName"`
	HasConflict    bool              `json:"hasConflict"`
	ConflictFields map[string]string `json:"conflictFields"`
	SystemState    *Service          `json:"systemState"`
}

// SyncOptions represents options for syncing configuration
type SyncOptions struct {
	Direction string   `json:"direction"` // collection_to_file or file_to_collection
	ItemIDs   []string `json:"itemIds,omitempty"`
}
