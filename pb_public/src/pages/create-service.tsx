import { useState, useEffect } from 'preact/hooks';
import { route } from 'preact-router';
import { api } from '../services/api';

interface FormData {
  name: string;
  port: string;
  version: string;
  subdomain: string;
  autoPort: boolean;
}

interface FormErrors {
  name?: string;
  port?: string;
  version?: string;
  subdomain?: string;
}

export function CreateService() {
  const [formData, setFormData] = useState<FormData>({
    name: '',
    port: '8091',
    version: 'latest',
    subdomain: '',
    autoPort: true,
  });
  
  const [errors, setErrors] = useState<FormErrors>({});
  const [loading, setLoading] = useState(false);
  const [versions, setVersions] = useState<string[]>([]);
  const [loadingVersions, setLoadingVersions] = useState(false);
  const [nextAvailablePort, setNextAvailablePort] = useState<number | null>(null);
  
  useEffect(() => {
    loadVersions();
    loadNextAvailablePort();
  }, []);
  
  useEffect(() => {
    // Auto-generate subdomain from name
    if (formData.name && !formData.subdomain) {
      setFormData(prev => ({
        ...prev,
        subdomain: prev.name.toLowerCase().replace(/[^a-z0-9]/g, '-')
      }));
    }
    
    // Set port to next available if autoPort is true
    if (formData.autoPort && nextAvailablePort) {
      setFormData(prev => ({
        ...prev,
        port: nextAvailablePort.toString()
      }));
    }
  }, [formData.name, formData.autoPort, nextAvailablePort]);
  
  async function loadVersions() {
    try {
      setLoadingVersions(true);
      const data = await api.getPocketBaseVersions();
      setVersions(data.versions || ['latest']);
    } catch (err) {
      console.error('Failed to load PocketBase versions:', err);
      // Default to latest if we can't load versions
      setVersions(['latest']);
    } finally {
      setLoadingVersions(false);
    }
  }
  
  async function loadNextAvailablePort() {
    try {
      const data = await api.getAvailablePorts();
      setNextAvailablePort(data.next_available_port || 8091);
    } catch (err) {
      console.error('Failed to load next available port:', err);
      setNextAvailablePort(8091);
    }
  }
  
  function handleChange(e: Event) {
    const target = e.target as HTMLInputElement;
    const name = target.name;
    const value = target.type === 'checkbox' ? target.checked : target.value;
    
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
    
    // Clear error when field is edited
    if (errors[name as keyof FormErrors]) {
      setErrors(prev => ({
        ...prev,
        [name]: undefined
      }));
    }
  }
  
  function validateForm(): boolean {
    const newErrors: FormErrors = {};
    
    if (!formData.name) {
      newErrors.name = 'Service name is required';
    } else if (!/^[a-zA-Z0-9_-]+$/.test(formData.name)) {
      newErrors.name = 'Service name can only contain letters, numbers, underscores, and hyphens';
    }
    
    if (!formData.autoPort) {
      if (!formData.port) {
        newErrors.port = 'Port is required';
      } else if (isNaN(Number(formData.port))) {
        newErrors.port = 'Port must be a number';
      } else if (Number(formData.port) < 1024 || Number(formData.port) > 65535) {
        newErrors.port = 'Port must be between 1024 and 65535';
      }
    }
    
    if (!formData.subdomain) {
      newErrors.subdomain = 'Subdomain is required';
    } else if (!/^[a-z0-9-]+$/.test(formData.subdomain)) {
      newErrors.subdomain = 'Subdomain can only contain lowercase letters, numbers, and hyphens';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }
  
  async function handleSubmit(e: Event) {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    try {
      setLoading(true);
      
      const serviceData = {
        name: formData.name,
        port: formData.autoPort ? nextAvailablePort : Number(formData.port),
        version: formData.version,
        subdomain: formData.subdomain,
        status: 'creating'
      };
      
      const createdService = await api.createService(serviceData);
      
      // Navigate to the service detail page
      route(`/services/${createdService.id}`);
    } catch (err: any) {
      console.error('Failed to create service:', err);
      
      // Handle validation errors from the server
      if (err.data?.data) {
        const serverErrors: FormErrors = {};
        Object.entries(err.data.data).forEach(([key, value]) => {
          serverErrors[key as keyof FormErrors] = Array.isArray(value) ? value[0] : String(value);
        });
        setErrors(serverErrors);
      }
    } finally {
      setLoading(false);
    }
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
        <h1 class="text-3xl font-bold">Create New Service</h1>
      </div>
      
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <form onSubmit={handleSubmit}>
            <div class="form-control mb-4">
              <label class="label">
                <span class="label-text">Service Name</span>
              </label>
              <input
                type="text"
                name="name"
                placeholder="my-service"
                class={`input input-bordered ${errors.name ? 'input-error' : ''}`}
                value={formData.name}
                onChange={handleChange}
              />
              {errors.name && <span class="text-error text-sm mt-1">{errors.name}</span>}
              <label class="label">
                <span class="label-text-alt">Service name will be used for the systemd service and directory name</span>
              </label>
            </div>
            
            <div class="form-control mb-4">
              <label class="label">
                <span class="label-text">Subdomain</span>
              </label>
              <input
                type="text"
                name="subdomain"
                placeholder="my-service"
                class={`input input-bordered ${errors.subdomain ? 'input-error' : ''}`}
                value={formData.subdomain}
                onChange={handleChange}
              />
              {errors.subdomain && <span class="text-error text-sm mt-1">{errors.subdomain}</span>}
              <label class="label">
                <span class="label-text-alt">The subdomain for accessing your service (e.g., my-service.yourdomain.com)</span>
              </label>
            </div>
            
            <div class="form-control mb-4">
              <label class="label">
                <div class="flex items-center justify-between w-full">
                  <span class="label-text">Port</span>
                  <div class="form-control">
                    <label class="label cursor-pointer">
                      <span class="label-text mr-2">Auto-assign port</span> 
                      <input 
                        type="checkbox" 
                        name="autoPort"
                        class="toggle toggle-primary" 
                        checked={formData.autoPort} 
                        onChange={handleChange}
                      />
                    </label>
                  </div>
                </div>
              </label>
              <input
                type="number"
                name="port"
                placeholder="8091"
                class={`input input-bordered ${errors.port ? 'input-error' : ''}`}
                value={formData.port}
                onChange={handleChange}
                disabled={formData.autoPort}
              />
              {errors.port && <span class="text-error text-sm mt-1">{errors.port}</span>}
              <label class="label">
                <span class="label-text-alt">
                  {formData.autoPort 
                    ? `Next available port: ${nextAvailablePort || 'Loading...'}`
                    : 'The port your PocketBase service will run on'}
                </span>
              </label>
            </div>
            
            <div class="form-control mb-6">
              <label class="label">
                <span class="label-text">PocketBase Version</span>
              </label>
              <select
                name="version"
                class={`select select-bordered ${errors.version ? 'select-error' : ''}`}
                value={formData.version}
                onChange={handleChange}
                disabled={loadingVersions}
              >
                {loadingVersions ? (
                  <option value="">Loading versions...</option>
                ) : (
                  versions.map(version => (
                    <option key={version} value={version}>{version}</option>
                  ))
                )}
              </select>
              {errors.version && <span class="text-error text-sm mt-1">{errors.version}</span>}
            </div>
            
            <div class="card-actions justify-end">
              <button 
                type="button" 
                class="btn btn-ghost" 
                onClick={() => route('/')}
                disabled={loading}
              >
                Cancel
              </button>
              <button 
                type="submit" 
                class="btn btn-primary" 
                disabled={loading}
              >
                {loading ? (
                  <>
                    <span class="loading loading-spinner loading-sm"></span>
                    Creating...
                  </>
                ) : 'Create Service'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}