import { useState, useEffect } from 'preact/hooks';
import { api } from '../services/api';
import { PocketBaseVersion, PocketBaseVersionsResponse, UpdateProgressEvent } from '../types/pocketbase-version';
import { AppError, createErrorFromException, getUserFriendlyErrorMessage } from '../types/error';
import { ErrorDisplay } from './error-display';

interface VersionManagerProps {
  serviceId: string;
  currentVersion: string;
  onVersionUpdate?: () => void;
}

export function VersionManager({ serviceId, currentVersion, onVersionUpdate }: VersionManagerProps) {
  const [versions, setVersions] = useState<PocketBaseVersion[]>([]);
  const [autoUpdateEnabled, setAutoUpdateEnabled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<AppError | null>(null);
  const [selectedVersion, setSelectedVersion] = useState<string>('');
  const [updateInProgress, setUpdateInProgress] = useState(false);
  const [updateProgress, setUpdateProgress] = useState<UpdateProgressEvent | null>(null);

  useEffect(() => {
    loadVersions();
  }, [serviceId]);

  async function loadVersions() {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getPocketBaseVersions() as PocketBaseVersionsResponse;
      setVersions(data.versions || []);
      setAutoUpdateEnabled(data.auto_update_enabled || false);
      setSelectedVersion('');
    } catch (err) {
      console.error('Failed to load PocketBase versions:', err);
      setError(createErrorFromException(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleUpdateVersion() {
    if (!selectedVersion || selectedVersion === currentVersion) return;
    
    try {
      setUpdateInProgress(true);
      setUpdateProgress({
        status: 'downloading',
        progress: 0,
        message: 'Starting download...'
      });
      
      // Simulate progress updates (in a real app, this would come from WebSockets or polling)
      const progressInterval = setInterval(() => {
        setUpdateProgress(prev => {
          if (!prev) return null;
          
          if (prev.status === 'downloading' && (prev.progress || 0) < 100) {
            return {
              ...prev,
              progress: (prev.progress || 0) + 10,
              message: `Downloading PocketBase ${selectedVersion}...`
            };
          } else if (prev.status === 'downloading' && (prev.progress || 0) >= 100) {
            return {
              status: 'extracting',
              progress: 0,
              message: 'Extracting files...'
            };
          } else if (prev.status === 'extracting' && (prev.progress || 0) < 100) {
            return {
              ...prev,
              progress: (prev.progress || 0) + 20,
              message: 'Extracting PocketBase binary...'
            };
          } else if (prev.status === 'extracting' && (prev.progress || 0) >= 100) {
            return {
              status: 'configuring',
              progress: 50,
              message: 'Configuring service...'
            };
          } else if (prev.status === 'configuring') {
            return {
              status: 'restarting',
              progress: 75,
              message: 'Restarting service...'
            };
          } else if (prev.status === 'restarting') {
            clearInterval(progressInterval);
            return {
              status: 'completed',
              progress: 100,
              message: 'Update completed successfully!'
            };
          }
          
          return prev;
        });
      }, 500);
      
      // Actual API call to update the PocketBase version
      await api.updatePocketBase(serviceId, selectedVersion);
      
      // Clear the interval if it's still running
      clearInterval(progressInterval);
      
      // Set final success state
      setUpdateProgress({
        status: 'completed',
        progress: 100,
        message: 'Update completed successfully!'
      });
      
      // Notify parent component that version was updated
      if (onVersionUpdate) {
        onVersionUpdate();
      }
      
      // Reset after a delay
      setTimeout(() => {
        setUpdateInProgress(false);
        setUpdateProgress(null);
        loadVersions();
      }, 3000);
      
    } catch (err) {
      console.error('Failed to update PocketBase version:', err);
      const appError = createErrorFromException(err);
      
      setUpdateProgress({
        status: 'failed',
        message: getUserFriendlyErrorMessage(appError)
      });
      
      // Reset after a delay
      setTimeout(() => {
        setUpdateInProgress(false);
        setUpdateProgress(null);
      }, 3000);
    }
  }

  async function toggleAutoUpdate() {
    try {
      // This would be implemented in a real app
      // await api.toggleAutoUpdate(serviceId, !autoUpdateEnabled);
      setAutoUpdateEnabled(!autoUpdateEnabled);
    } catch (err) {
      console.error('Failed to toggle auto-update:', err);
    }
  }

  if (loading) {
    return (
      <div class="flex justify-center my-8">
        <span class="loading loading-spinner loading-md"></span>
      </div>
    );
  }

  if (error) {
    return <ErrorDisplay error={error} onRetry={loadVersions} />;
  }

  return (
    <div>
      <div class="flex flex-col md:flex-row gap-4 mb-6">
        <div class="form-control flex-grow">
          <label class="label">
            <span class="label-text">Current Version</span>
          </label>
          <div class="input-group">
            <span class="bg-base-300 px-4 py-2 flex items-center">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            </span>
            <input 
              type="text" 
              class="input input-bordered flex-grow" 
              value={currentVersion} 
              disabled 
            />
          </div>
        </div>
        
        <div class="form-control flex-grow">
          <label class="label">
            <span class="label-text">Available Versions</span>
          </label>
          <div class="input-group">
            <span class="bg-base-300 px-4 py-2 flex items-center">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
            </span>
            <select 
              class="select select-bordered flex-grow" 
              value={selectedVersion}
              onChange={(e) => setSelectedVersion((e.target as HTMLSelectElement).value)}
              disabled={updateInProgress}
            >
              <option value="">Select a version</option>
              {versions.map(version => (
                <option 
                  key={version.version} 
                  value={version.version}
                  disabled={version.version === currentVersion}
                >
                  {version.version} {version.is_latest ? '(Latest)' : ''} - {new Date(version.release_date).toLocaleDateString()}
                </option>
              ))}
            </select>
            <button 
              class="btn btn-primary" 
              disabled={!selectedVersion || selectedVersion === currentVersion || updateInProgress}
              onClick={handleUpdateVersion}
            >
              {updateInProgress ? (
                <>
                  <span class="loading loading-spinner loading-sm"></span>
                  Updating...
                </>
              ) : 'Update'}
            </button>
          </div>
        </div>
      </div>
      
      {updateProgress && (
        <div class={`alert ${updateProgress.status === 'failed' ? 'alert-error' : updateProgress.status === 'completed' ? 'alert-success' : 'alert-info'} mb-6`}>
          <div class="flex flex-col w-full">
            <div class="flex items-center">
              {updateProgress.status === 'completed' ? (
                <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              ) : updateProgress.status === 'failed' ? (
                <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              ) : (
                <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              )}
              <span class="ml-2">{updateProgress.message}</span>
            </div>
            
            {updateProgress.progress !== undefined && updateProgress.status !== 'completed' && updateProgress.status !== 'failed' && (
              <progress 
                class="progress progress-primary w-full mt-2" 
                value={updateProgress.progress} 
                max="100"
              ></progress>
            )}
          </div>
        </div>
      )}
      
      <div class="divider"></div>
      
      <div class="flex justify-between items-center">
        <div>
          <h3 class="font-bold">Automatic Updates</h3>
          <p class="text-sm opacity-70">When enabled, this service will automatically update to the latest PocketBase version when available.</p>
        </div>
        <div class="form-control">
          <label class="label cursor-pointer">
            <input 
              type="checkbox" 
              class="toggle toggle-primary" 
              checked={autoUpdateEnabled}
              onChange={toggleAutoUpdate}
            />
          </label>
        </div>
      </div>
      
      <div class="mt-6">
        <h3 class="font-bold mb-2">Version History</h3>
        <div class="overflow-x-auto">
          <table class="table table-zebra">
            <thead>
              <tr>
                <th>Version</th>
                <th>Release Date</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {versions.slice(0, 5).map(version => (
                <tr key={version.version}>
                  <td>{version.version}</td>
                  <td>{new Date(version.release_date).toLocaleDateString()}</td>
                  <td>
                    {version.version === currentVersion ? (
                      <span class="badge badge-primary">Current</span>
                    ) : version.is_latest ? (
                      <span class="badge badge-accent">Latest</span>
                    ) : (
                      <span class="badge">Previous</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}