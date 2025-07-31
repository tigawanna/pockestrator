import { useState, useEffect } from 'preact/hooks';
import { route } from 'preact-router';
import { api } from '../services/api';
import { Service, ServiceValidation } from '../types/service';
import { ConfigItem, ConfigSyncResponse } from '../types/config-sync';
import { StatusBadge } from '../components/status-badge';
import { ValidationIndicator } from '../components/validation-indicator';
import { ConfigSyncItem } from '../components/config-sync-item';
import { VersionManager } from '../components/version-manager';
import { FileManager } from '../components/file-manager';

interface ServiceDetailProps {
  id?: string;
}

export function ServiceDetail(props: ServiceDetailProps) {
  const [service, setService] = useState<Service | null>(null);
  const [validation, setValidation] = useState<ServiceValidation | null>(null);
  const [logs, setLogs] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [validationLoading, setValidationLoading] = useState(false);
  const [logsLoading, setLogsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState('overview');
  const [actionLoading, setActionLoading] = useState<Record<string, boolean>>({});
  // Configuration sync state
  const [configItems, setConfigItems] = useState<ConfigItem[]>([]);
  const [configSyncLoading, setConfigSyncLoading] = useState(false);
  const [configSyncError, setConfigSyncError] = useState<string | null>(null);

  useEffect(() => {
    if (props.id) {
      loadService(props.id);
    }
  }, [props.id]);
  
  useEffect(() => {
    if (props.id && activeTab === 'configuration') {
      loadConfigSyncStatus(props.id);
    }
  }, [props.id, activeTab]);

  async function loadService(id: string) {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getService(id);
      setService(data);
      validateService(id);
      loadServiceLogs(id);
    } catch (err) {
      console.error('Failed to load service:', err);
      setError('Failed to load service. Please try again.');
    } finally {
      setLoading(false);
    }
  }

  async function validateService(id: string) {
    try {
      setValidationLoading(true);
      const data = await api.validateService(id);
      setValidation(data);
    } catch (err) {
      console.error('Failed to validate service:', err);
    } finally {
      setValidationLoading(false);
    }
  }

  async function loadServiceLogs(id: string) {
    try {
      setLogsLoading(true);
      const data = await api.getServiceLogs(id);
      setLogs(data.logs || 'No logs available');
    } catch (err) {
      console.error('Failed to load service logs:', err);
      setLogs('Failed to load logs');
    } finally {
      setLogsLoading(false);
    }
  }

  // Function removed - now handled by FileManager component

  async function handleRestartService() {
    if (!props.id) return;
    
    try {
      setActionLoading(prev => ({ ...prev, restart: true }));
      await api.restartService(props.id);
      // Refresh validation and logs after restart
      setTimeout(() => {
        validateService(props.id!);
        loadServiceLogs(props.id!);
      }, 2000);
    } catch (err) {
      console.error('Failed to restart service:', err);
    } finally {
      setActionLoading(prev => ({ ...prev, restart: false }));
    }
  }

  async function handleSyncConfig() {
    if (!props.id) return;
    
    try {
      setActionLoading(prev => ({ ...prev, sync: true }));
      await api.syncConfig(props.id);
      // Refresh validation after sync
      validateService(props.id);
      // Refresh config sync status if we're on the configuration tab
      if (activeTab === 'configuration') {
        loadConfigSyncStatus(props.id);
      }
    } catch (err) {
      console.error('Failed to sync configuration:', err);
    } finally {
      setActionLoading(prev => ({ ...prev, sync: false }));
    }
  }
  
  async function loadConfigSyncStatus(id: string) {
    try {
      setConfigSyncLoading(true);
      setConfigSyncError(null);
      const data = await api.getConfigSyncStatus(id) as ConfigSyncResponse;
      setConfigItems(data.items || []);
    } catch (err) {
      console.error('Failed to load configuration sync status:', err);
      setConfigSyncError('Failed to load configuration sync status. Please try again.');
    } finally {
      setConfigSyncLoading(false);
    }
  }
  
  async function handleSyncConfigItem(itemId: string, direction: 'collection_to_file' | 'file_to_collection') {
    if (!props.id) return;
    
    try {
      await api.syncConfigItem(props.id, {
        direction,
        itemIds: [itemId]
      });
      
      // Refresh config sync status
      loadConfigSyncStatus(props.id);
      
      // Also refresh validation
      validateService(props.id);
    } catch (err) {
      console.error('Failed to sync configuration item:', err);
    }
  }

  // File management functions removed - now handled by FileManager component

  if (loading) {
    return (
      <div class="flex justify-center my-12">
        <span class="loading loading-spinner loading-lg"></span>
      </div>
    );
  }

  if (error || !service) {
    return (
      <div class="alert alert-error mb-6">
        <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span>{error || 'Service not found'}</span>
        <button class="btn btn-sm" onClick={() => route('/')}>Back to Dashboard</button>
      </div>
    );
  }

  return (
    <div>
      <div class="flex items-center mb-6">
        <button 
          class="btn btn-ghost btn-sm mr-2" 
          onClick={() => route('/')}
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
        </button>
        <h1 class="text-3xl font-bold">{service.name}</h1>
        <div class="ml-4 flex items-center gap-2">
          <StatusBadge status={service.status} />
          <ValidationIndicator validation={validation} loading={validationLoading} />
        </div>
      </div>

      <div class="tabs tabs-boxed mb-6">
        <a 
          class={`tab ${activeTab === 'overview' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          Overview
        </a>
        <a 
          class={`tab ${activeTab === 'files' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('files')}
        >
          Files
        </a>
        <a 
          class={`tab ${activeTab === 'logs' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('logs')}
        >
          Logs
        </a>
        <a 
          class={`tab ${activeTab === 'configuration' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('configuration')}
        >
          Configuration
        </a>
        <a 
          class={`tab ${activeTab === 'versions' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('versions')}
        >
          Versions
        </a>
      </div>

      {activeTab === 'overview' && (
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div class="card bg-base-100 shadow-xl">
            <div class="card-body">
              <h2 class="card-title">Service Information</h2>
              <div class="overflow-x-auto">
                <table class="table">
                  <tbody>
                    <tr>
                      <td class="font-bold">Name</td>
                      <td>{service.name}</td>
                    </tr>
                    <tr>
                      <td class="font-bold">Port</td>
                      <td>{service.port}</td>
                    </tr>
                    <tr>
                      <td class="font-bold">Subdomain</td>
                      <td>{service.subdomain}</td>
                    </tr>
                    <tr>
                      <td class="font-bold">Version</td>
                      <td>{service.version}</td>
                    </tr>
                    <tr>
                      <td class="font-bold">Status</td>
                      <td><StatusBadge status={service.status} /></td>
                    </tr>
                    <tr>
                      <td class="font-bold">Created</td>
                      <td>{new Date(service.created).toLocaleString()}</td>
                    </tr>
                    <tr>
                      <td class="font-bold">Updated</td>
                      <td>{new Date(service.updated).toLocaleString()}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div class="card-actions justify-end mt-4">
                <button 
                  class="btn btn-primary" 
                  onClick={handleRestartService}
                  disabled={actionLoading.restart}
                >
                  {actionLoading.restart ? (
                    <>
                      <span class="loading loading-spinner loading-sm"></span>
                      Restarting...
                    </>
                  ) : 'Restart Service'}
                </button>
              </div>
            </div>
          </div>

          <div class="card bg-base-100 shadow-xl">
            <div class="card-body">
              <h2 class="card-title">Validation Status</h2>
              {validationLoading ? (
                <div class="flex justify-center my-8">
                  <span class="loading loading-spinner loading-md"></span>
                </div>
              ) : validation ? (
                <div>
                  <div class="overflow-x-auto">
                    <table class="table">
                      <tbody>
                        <tr>
                          <td class="font-bold">Systemd Service</td>
                          <td>
                            {validation.systemd_exists ? (
                              <span class="badge badge-success">Exists</span>
                            ) : (
                              <span class="badge badge-error">Missing</span>
                            )}
                          </td>
                        </tr>
                        <tr>
                          <td class="font-bold">Service Running</td>
                          <td>
                            {validation.systemd_running ? (
                              <span class="badge badge-success">Running</span>
                            ) : (
                              <span class="badge badge-error">Stopped</span>
                            )}
                          </td>
                        </tr>
                        <tr>
                          <td class="font-bold">Caddy Config</td>
                          <td>
                            {validation.caddy_configured ? (
                              <span class="badge badge-success">Configured</span>
                            ) : (
                              <span class="badge badge-error">Missing</span>
                            )}
                          </td>
                        </tr>
                        <tr>
                          <td class="font-bold">PocketBase Binary</td>
                          <td>
                            {validation.binary_exists ? (
                              <span class="badge badge-success">Exists</span>
                            ) : (
                              <span class="badge badge-error">Missing</span>
                            )}
                          </td>
                        </tr>
                        <tr>
                          <td class="font-bold">Port Configuration</td>
                          <td>
                            {validation.port_matches ? (
                              <span class="badge badge-success">Matches</span>
                            ) : (
                              <span class="badge badge-error">Mismatch</span>
                            )}
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>

                  {validation.issues.length > 0 && (
                    <div class="mt-4">
                      <h3 class="font-bold mb-2">Issues:</h3>
                      <ul class="list-disc list-inside">
                        {validation.issues.map((issue, index) => (
                          <li key={index} class="text-error">{issue}</li>
                        ))}
                      </ul>
                    </div>
                  )}

                  <div class="card-actions justify-end mt-4">
                    <button 
                      class="btn btn-outline btn-primary" 
                      onClick={() => validateService(props.id!)}
                      disabled={validationLoading}
                    >
                      {validationLoading ? (
                        <>
                          <span class="loading loading-spinner loading-sm"></span>
                          Validating...
                        </>
                      ) : 'Refresh Validation'}
                    </button>
                    <button 
                      class="btn btn-primary" 
                      onClick={handleSyncConfig}
                      disabled={actionLoading.sync}
                    >
                      {actionLoading.sync ? (
                        <>
                          <span class="loading loading-spinner loading-sm"></span>
                          Syncing...
                        </>
                      ) : 'Sync Configuration'}
                    </button>
                  </div>
                </div>
              ) : (
                <div class="alert alert-info">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-current shrink-0 w-6 h-6">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                  </svg>
                  <span>No validation data available</span>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {activeTab === 'files' && (
        <div class="card bg-base-100 shadow-xl">
          <div class="card-body">
            <FileManager 
              serviceId={props.id!} 
              onFileChange={() => {
                // Refresh validation after file changes
                validateService(props.id!);
              }}
            />
          </div>
        </div>
      )}

      {activeTab === 'logs' && (
        <div class="card bg-base-100 shadow-xl">
          <div class="card-body">
            <div class="flex justify-between items-center">
              <h2 class="card-title">Service Logs</h2>
              <button 
                class="btn btn-sm btn-outline" 
                onClick={() => loadServiceLogs(props.id!)}
                disabled={logsLoading}
              >
                {logsLoading ? (
                  <>
                    <span class="loading loading-spinner loading-sm"></span>
                    Refreshing...
                  </>
                ) : 'Refresh Logs'}
              </button>
            </div>
            
            {logsLoading ? (
              <div class="flex justify-center my-8">
                <span class="loading loading-spinner loading-md"></span>
              </div>
            ) : (
              <div class="mockup-code mt-4 max-h-96 overflow-auto">
                <pre><code>{logs}</code></pre>
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'versions' && (
        <div class="card bg-base-100 shadow-xl">
          <div class="card-body">
            <h2 class="card-title">PocketBase Version Management</h2>
            <VersionManager 
              serviceId={props.id!} 
              currentVersion={service.version}
              onVersionUpdate={() => {
                // Refresh service data and validation after version update
                loadService(props.id!);
              }}
            />
          </div>
        </div>
      )}

      {activeTab === 'configuration' && (
        <div class="card bg-base-100 shadow-xl">
          <div class="card-body">
            <div class="flex justify-between items-center mb-4">
              <h2 class="card-title">Configuration Management</h2>
              <button 
                class="btn btn-sm btn-outline" 
                onClick={() => loadConfigSyncStatus(props.id!)}
                disabled={configSyncLoading}
              >
                {configSyncLoading ? (
                  <>
                    <span class="loading loading-spinner loading-sm"></span>
                    Refreshing...
                  </>
                ) : 'Refresh Status'}
              </button>
            </div>
            
            <div class="mb-6">
              <p class="mb-2">
                This page shows the synchronization status between the configuration stored in the database 
                and the actual system files. When conflicts are detected, you can choose to sync in either direction.
              </p>
              
              <div class="flex flex-wrap gap-2 mt-4">
                <div class="badge badge-success gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="inline-block w-4 h-4 stroke-current">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                  </svg>
                  Synced
                </div>
                <div class="badge badge-warning gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="inline-block w-4 h-4 stroke-current">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
                  </svg>
                  Conflict
                </div>
                <div class="badge badge-error gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="inline-block w-4 h-4 stroke-current">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                  </svg>
                  Missing
                </div>
              </div>
            </div>
            
            {configSyncLoading ? (
              <div class="flex justify-center my-8">
                <span class="loading loading-spinner loading-lg"></span>
              </div>
            ) : configSyncError ? (
              <div class="alert alert-error">
                <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{configSyncError}</span>
                <button 
                  class="btn btn-sm" 
                  onClick={() => loadConfigSyncStatus(props.id!)}
                >
                  Retry
                </button>
              </div>
            ) : configItems.length === 0 ? (
              <div class="alert alert-info">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-current shrink-0 w-6 h-6">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
                <span>No configuration items found. Try refreshing the status.</span>
              </div>
            ) : (
              <div>
                <div class="flex justify-between items-center mb-4">
                  <div class="flex items-center gap-2">
                    <h3 class="font-bold">Configuration Items</h3>
                    <div class="badge">{configItems.length}</div>
                  </div>
                  
                  <div class="dropdown dropdown-end">
                    <label tabIndex={0} class="btn btn-sm">
                      Sync All
                    </label>
                    <ul tabIndex={0} class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52">
                      <li>
                        <a onClick={() => {
                          if (props.id) {
                            api.syncConfigItem(props.id, { direction: 'collection_to_file' })
                              .then(() => {
                                loadConfigSyncStatus(props.id!);
                                validateService(props.id!);
                              });
                          }
                        }}>
                          All Collection → File
                        </a>
                      </li>
                      <li>
                        <a onClick={() => {
                          if (props.id) {
                            api.syncConfigItem(props.id, { direction: 'file_to_collection' })
                              .then(() => {
                                loadConfigSyncStatus(props.id!);
                                validateService(props.id!);
                              });
                          }
                        }}>
                          All File → Collection
                        </a>
                      </li>
                    </ul>
                  </div>
                </div>
                
                <div class="space-y-4">
                  {configItems.map((item) => (
                    <ConfigSyncItem 
                      key={item.id} 
                      item={item} 
                      onSync={handleSyncConfigItem} 
                    />
                  ))}
                </div>
              </div>
            )}
            
            <div class="card-actions justify-end mt-6">
              <button 
                class="btn btn-primary" 
                onClick={handleSyncConfig}
                disabled={actionLoading.sync}
              >
                {actionLoading.sync ? (
                  <>
                    <span class="loading loading-spinner loading-sm"></span>
                    Syncing All...
                  </>
                ) : 'Sync All Configuration'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}