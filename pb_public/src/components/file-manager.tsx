import { useState, useEffect } from 'preact/hooks';
import { api } from '../services/api';

interface FileManagerProps {
  serviceId: string;
  onFileChange?: () => void;
}

interface FileUploadState {
  directory: string;
  files: File[];
  uploading: boolean;
  error: string | null;
  dragActive: boolean;
}

interface DirectoryFilesState {
  [key: string]: string[];
}

interface LoadingState {
  [key: string]: boolean;
}

export function FileManager({ serviceId, onFileChange }: FileManagerProps) {
  const [fileUpload, setFileUpload] = useState<FileUploadState>({
    directory: 'pb_public',
    files: [],
    uploading: false,
    error: null,
    dragActive: false
  });
  
  const [directoryFiles, setDirectoryFiles] = useState<DirectoryFilesState>({
    pb_public: [],
    pb_migrations: [],
    pb_hooks: []
  });
  
  const [loadingFiles, setLoadingFiles] = useState<LoadingState>({
    pb_public: false,
    pb_migrations: false,
    pb_hooks: false
  });

  const directories = ['pb_public', 'pb_migrations', 'pb_hooks'];

  useEffect(() => {
    if (serviceId) {
      loadDirectoryFiles(serviceId, fileUpload.directory);
    }
  }, [serviceId, fileUpload.directory]);

  async function loadDirectoryFiles(id: string, directory: string) {
    try {
      setLoadingFiles(prev => ({ ...prev, [directory]: true }));
      const data = await api.getFiles(id, directory);
      setDirectoryFiles(prev => ({ 
        ...prev, 
        [directory]: data.files || [] 
      }));
    } catch (err) {
      console.error(`Failed to load ${directory} files:`, err);
    } finally {
      setLoadingFiles(prev => ({ ...prev, [directory]: false }));
    }
  }

  function handleFileChange(e: Event) {
    const target = e.target as HTMLInputElement;
    if (target.files) {
      setFileUpload(prev => ({
        ...prev,
        files: Array.from(target.files || []),
        error: null
      }));
    }
  }

  function handleDirectoryChange(e: Event) {
    const target = e.target as HTMLSelectElement;
    setFileUpload(prev => ({
      ...prev,
      directory: target.value,
      error: null
    }));
  }

  async function handleFileUpload() {
    if (!serviceId || fileUpload.files.length === 0) return;
    
    try {
      setFileUpload(prev => ({ ...prev, uploading: true, error: null }));
      
      // Upload each file
      for (const file of fileUpload.files) {
        await api.uploadFile(serviceId, fileUpload.directory, file);
      }
      
      // Refresh file list
      loadDirectoryFiles(serviceId, fileUpload.directory);
      
      // Clear file selection
      setFileUpload(prev => ({
        ...prev,
        files: [],
        uploading: false
      }));

      // Notify parent component if needed
      if (onFileChange) {
        onFileChange();
      }
    } catch (err) {
      console.error('Failed to upload files:', err);
      setFileUpload(prev => ({
        ...prev,
        uploading: false,
        error: 'Failed to upload files. Please try again.'
      }));
    }
  }

  async function handleDeleteFile(directory: string, filename: string) {
    if (!serviceId) return;
    
    if (!confirm(`Are you sure you want to delete ${filename}?`)) {
      return;
    }
    
    try {
      await api.deleteFile(serviceId, directory, filename);
      // Refresh file list
      loadDirectoryFiles(serviceId, directory);
      
      // Notify parent component if needed
      if (onFileChange) {
        onFileChange();
      }
    } catch (err) {
      console.error('Failed to delete file:', err);
    }
  }

  // Drag and drop handlers
  function handleDragEnter(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    setFileUpload(prev => ({ ...prev, dragActive: true }));
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    setFileUpload(prev => ({ ...prev, dragActive: false }));
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    setFileUpload(prev => ({ ...prev, dragActive: false }));
    
    if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
      setFileUpload(prev => ({
        ...prev,
        files: Array.from(e.dataTransfer?.files || []),
        error: null
      }));
    }
  }

  return (
    <div>
      <h2 class="card-title mb-4">File Management</h2>
      
      <div class="mb-6">
        <h3 class="font-bold mb-2">Upload Files</h3>
        <div 
          class={`border-2 border-dashed rounded-lg p-6 mb-4 transition-colors ${
            fileUpload.dragActive ? 'border-primary bg-primary/10' : 'border-base-300'
          }`}
          onDragEnter={handleDragEnter}
          onDragLeave={handleDragLeave}
          onDragOver={handleDragOver}
          onDrop={handleDrop}
        >
          <div class="flex flex-col md:flex-row gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">Directory</span>
              </label>
              <select 
                class="select select-bordered" 
                value={fileUpload.directory}
                onChange={handleDirectoryChange}
                disabled={fileUpload.uploading}
              >
                {directories.map(dir => (
                  <option key={dir} value={dir}>{dir}</option>
                ))}
              </select>
            </div>
            
            <div class="form-control flex-grow">
              <label class="label">
                <span class="label-text">Files</span>
              </label>
              <input 
                type="file" 
                class="file-input file-input-bordered w-full" 
                multiple
                onChange={handleFileChange}
                disabled={fileUpload.uploading}
              />
            </div>
            
            <div class="form-control">
              <label class="label">
                <span class="label-text">&nbsp;</span>
              </label>
              <button 
                class="btn btn-primary" 
                onClick={handleFileUpload}
                disabled={fileUpload.uploading || fileUpload.files.length === 0}
              >
                {fileUpload.uploading ? (
                  <>
                    <span class="loading loading-spinner loading-sm"></span>
                    Uploading...
                  </>
                ) : 'Upload'}
              </button>
            </div>
          </div>
          
          {fileUpload.dragActive && (
            <div class="text-center mt-4 text-primary">
              <p>Drop files here to upload</p>
            </div>
          )}
          
          {!fileUpload.dragActive && fileUpload.files.length === 0 && (
            <div class="text-center mt-4 text-base-content/70">
              <p>Drag and drop files here or use the file selector above</p>
            </div>
          )}
        </div>
        
        {fileUpload.files.length > 0 && (
          <div class="mt-2 p-4 bg-base-200 rounded-lg">
            <p class="font-bold">Selected {fileUpload.files.length} file(s):</p>
            <ul class="list-disc list-inside mt-2">
              {Array.from(fileUpload.files).map((file, index) => (
                <li key={index} class="text-sm">
                  {file.name} ({(file.size / 1024).toFixed(1)} KB)
                </li>
              ))}
            </ul>
          </div>
        )}
        
        {fileUpload.error && (
          <div class="alert alert-error mt-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{fileUpload.error}</span>
          </div>
        )}
      </div>
      
      <div class="divider"></div>
      
      <div class="tabs tabs-boxed mb-4">
        {directories.map(dir => (
          <a 
            key={dir}
            class={`tab ${fileUpload.directory === dir ? 'tab-active' : ''}`}
            onClick={() => setFileUpload(prev => ({ ...prev, directory: dir }))}
          >
            {dir}
          </a>
        ))}
      </div>
      
      <h3 class="font-bold mb-2">Files in {fileUpload.directory}</h3>
      
      {loadingFiles[fileUpload.directory] ? (
        <div class="flex justify-center my-8">
          <span class="loading loading-spinner loading-md"></span>
        </div>
      ) : directoryFiles[fileUpload.directory]?.length === 0 ? (
        <div class="alert alert-info">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-current shrink-0 w-6 h-6">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          </svg>
          <span>No files found in {fileUpload.directory}</span>
        </div>
      ) : (
        <div class="overflow-x-auto">
          <table class="table table-zebra w-full">
            <thead>
              <tr>
                <th>Filename</th>
                <th class="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {directoryFiles[fileUpload.directory]?.map((filename, index) => (
                <tr key={index}>
                  <td class="break-all">{filename}</td>
                  <td class="text-right">
                    <button 
                      class="btn btn-sm btn-error"
                      onClick={() => handleDeleteFile(fileUpload.directory, filename)}
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      
      <div class="card-actions justify-end mt-4">
        <button 
          class="btn btn-outline" 
          onClick={() => loadDirectoryFiles(serviceId, fileUpload.directory)}
          disabled={loadingFiles[fileUpload.directory]}
        >
          {loadingFiles[fileUpload.directory] ? (
            <>
              <span class="loading loading-spinner loading-sm"></span>
              Refreshing...
            </>
          ) : (
            <>
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Refresh Files
            </>
          )}
        </button>
      </div>
    </div>
  );
}