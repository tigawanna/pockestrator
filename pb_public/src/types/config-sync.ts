export interface ConfigItem {
  id: string;
  name: string;
  type: 'systemd' | 'caddy' | 'binary';
  collectionValue: string;
  fileValue: string;
  status: 'synced' | 'conflict' | 'missing_file' | 'missing_collection';
}

export interface ConfigSyncResponse {
  items: ConfigItem[];
  timestamp: string;
}

export interface SyncOptions {
  direction: 'collection_to_file' | 'file_to_collection';
  itemIds?: string[]; // If not provided, sync all items
}