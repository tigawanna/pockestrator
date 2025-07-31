export interface Service {
  id: string;
  name: string;
  port: number;
  version: string;
  subdomain: string;
  status: 'creating' | 'running' | 'stopped' | 'error';
  created: string;
  updated: string;
}

export interface ServiceValidation {
  systemd_exists: boolean;
  systemd_running: boolean;
  caddy_configured: boolean;
  binary_exists: boolean;
  port_matches: boolean;
  issues: string[];
}