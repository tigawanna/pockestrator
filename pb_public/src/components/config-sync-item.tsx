import { useState } from 'preact/hooks';
import { ConfigItem } from '../types/config-sync';

interface ConfigSyncItemProps {
  item: ConfigItem;
  onSync: (itemId: string, direction: 'collection_to_file' | 'file_to_collection') => Promise<void>;
}

export function ConfigSyncItem({ item, onSync }: ConfigSyncItemProps) {
  const [syncing, setSyncing] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const handleSync = async (direction: 'collection_to_file' | 'file_to_collection') => {
    try {
      setSyncing(true);
      await onSync(item.id, direction);
    } finally {
      setSyncing(false);
    }
  };

  const getStatusBadge = () => {
    switch (item.status) {
      case 'synced':
        return (
          <span className="badge badge-success gap-1">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" className="inline-block w-4 h-4 stroke-current">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7"></path>
            </svg>
            Synced
          </span>
        );
      case 'conflict':
        return (
          <span className="badge badge-warning gap-1">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" className="inline-block w-4 h-4 stroke-current">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
            </svg>
            Conflict
          </span>
        );
      case 'missing_file':
        return (
          <span className="badge badge-error gap-1">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" className="inline-block w-4 h-4 stroke-current">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
            Missing File
          </span>
        );
      case 'missing_collection':
        return (
          <span className="badge badge-error gap-1">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" className="inline-block w-4 h-4 stroke-current">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
            Missing in Collection
          </span>
        );
      default:
        return <span className="badge">Unknown</span>;
    }
  };

  const getTypeIcon = () => {
    switch (item.type) {
      case 'systemd':
        return (
          <div className="avatar placeholder">
            <div className="bg-neutral text-neutral-content rounded-full w-8">
              <span>SD</span>
            </div>
          </div>
        );
      case 'caddy':
        return (
          <div className="avatar placeholder">
            <div className="bg-primary text-primary-content rounded-full w-8">
              <span>CD</span>
            </div>
          </div>
        );
      case 'binary':
        return (
          <div className="avatar placeholder">
            <div className="bg-secondary text-secondary-content rounded-full w-8">
              <span>PB</span>
            </div>
          </div>
        );
      default:
        return (
          <div className="avatar placeholder">
            <div className="bg-base-300 rounded-full w-8">
              <span>?</span>
            </div>
          </div>
        );
    }
  };

  return (
    <div className="card bg-base-200 shadow-sm mb-4">
      <div className="card-body p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {getTypeIcon()}
            <div>
              <h3 className="font-bold">{item.name}</h3>
              <div className="text-sm opacity-70">{item.type}</div>
            </div>
            <div className="ml-4">{getStatusBadge()}</div>
          </div>
          
          <div className="flex items-center gap-2">
            {item.status !== 'synced' && (
              <div className="dropdown dropdown-end">
                <label tabIndex={0} className="btn btn-sm">
                  {syncing ? (
                    <span className="loading loading-spinner loading-xs"></span>
                  ) : (
                    "Sync"
                  )}
                </label>
                <ul tabIndex={0} className="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52">
                  <li>
                    <a onClick={() => handleSync('collection_to_file')}>
                      Collection → File
                    </a>
                  </li>
                  <li>
                    <a onClick={() => handleSync('file_to_collection')}>
                      File → Collection
                    </a>
                  </li>
                </ul>
              </div>
            )}
            
            <button 
              className="btn btn-sm btn-ghost"
              onClick={() => setExpanded(!expanded)}
            >
              {expanded ? (
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                </svg>
              ) : (
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              )}
            </button>
          </div>
        </div>
        
        {expanded && (
          <div className="mt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-base-100 p-3 rounded-lg">
              <div className="font-bold mb-2">Collection Value</div>
              <pre className="whitespace-pre-wrap text-sm overflow-auto max-h-48 bg-base-300 p-2 rounded">
                {item.collectionValue || 'No value in collection'}
              </pre>
            </div>
            
            <div className="bg-base-100 p-3 rounded-lg">
              <div className="font-bold mb-2">File Value</div>
              <pre className="whitespace-pre-wrap text-sm overflow-auto max-h-48 bg-base-300 p-2 rounded">
                {item.fileValue || 'No value in file'}
              </pre>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}