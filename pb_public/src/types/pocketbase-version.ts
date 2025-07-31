export interface PocketBaseVersion {
  version: string;
  release_date: string;
  is_latest: boolean;
}

export interface PocketBaseVersionsResponse {
  versions: PocketBaseVersion[];
  current_version: string;
  auto_update_enabled: boolean;
}

export interface UpdateProgressEvent {
  status: 'downloading' | 'extracting' | 'configuring' | 'restarting' | 'completed' | 'failed';
  progress?: number;
  message?: string;
}