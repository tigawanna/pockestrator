import { useEffect, useState } from 'preact/hooks';
import { route } from 'preact-router';
import { api } from '../services/api';
import { Service, ServiceValidation } from '../types/service';
import { StatusBadge } from '../components/status-badge';
import { ValidationIndicator } from '../components/validation-indicator';

export function Dashboard() {
  const [services, setServices] = useState<Service[]>([]);
  const [validations, setValidations] = useState<Record<string, ServiceValidation>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [validationLoading, setValidationLoading] = useState<Record<string, boolean>>({});

  useEffect(() => {
    loadServices();
  }, []);

  async function loadServices() {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getServices();
      setServices(data);
      
      // Start validation for each service
      data.forEach(service => {
        validateService(service.id);
      });
    } catch (err) {
      console.error('Failed to load services:', err);
      setError('Failed to load services. Please try again.');
    } finally {
      setLoading(false);
    }
  }

  async function validateService(id: string) {
    try {
      setValidationLoading(prev => ({ ...prev, [id]: true }));
      const validation = await api.validateService(id);
      setValidations(prev => ({ ...prev, [id]: validation }));
    } catch (err) {
      console.error(`Failed to validate service ${id}:`, err);
    } finally {
      setValidationLoading(prev => ({ ...prev, [id]: false }));
    }
  }

  async function handleRestartService(id: string) {
    try {
      await api.restartService(id);
      // Refresh validation after restart
      setTimeout(() => validateService(id), 2000);
    } catch (err) {
      console.error(`Failed to restart service ${id}:`, err);
    }
  }

  return (
    <div>
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold">Services Dashboard</h1>
        <button 
          class="btn btn-primary" 
          onClick={() => route('/services/new')}
        >
          Create New Service
        </button>
      </div>

      {error && (
        <div class="alert alert-error mb-6">
          <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>{error}</span>
          <button class="btn btn-sm" onClick={loadServices}>Retry</button>
        </div>
      )}

      {loading ? (
        <div class="flex justify-center my-12">
          <span class="loading loading-spinner loading-lg"></span>
        </div>
      ) : services.length === 0 ? (
        <div class="card bg-base-100 shadow-xl">
          <div class="card-body text-center">
            <h2 class="card-title justify-center">No Services Found</h2>
            <p>Get started by creating your first PocketBase service.</p>
            <div class="card-actions justify-center mt-4">
              <button 
                class="btn btn-primary" 
                onClick={() => route('/services/new')}
              >
                Create New Service
              </button>
            </div>
          </div>
        </div>
      ) : (
        <div class="grid grid-cols-1 gap-6">
          {services.map(service => (
            <div key={service.id} class="card bg-base-100 shadow-xl">
              <div class="card-body">
                <div class="flex justify-between items-start">
                  <div>
                    <h2 class="card-title">{service.name}</h2>
                    <div class="flex gap-2 items-center mt-1">
                      <StatusBadge status={service.status} />
                      <ValidationIndicator 
                        validation={validations[service.id]} 
                        loading={validationLoading[service.id]} 
                      />
                    </div>
                  </div>
                  <div class="dropdown dropdown-end">
                    <label tabIndex={0} class="btn btn-sm btn-ghost btn-circle">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z" />
                      </svg>
                    </label>
                    <ul tabIndex={0} class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52">
                      <li><a onClick={() => route(`/services/${service.id}`)}>View Details</a></li>
                      <li><a onClick={() => handleRestartService(service.id)}>Restart Service</a></li>
                      <li><a onClick={() => validateService(service.id)}>Validate Configuration</a></li>
                    </ul>
                  </div>
                </div>
                
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
                  <div class="stat bg-base-200 rounded-box p-4">
                    <div class="stat-title">Port</div>
                    <div class="stat-value text-xl">{service.port}</div>
                  </div>
                  <div class="stat bg-base-200 rounded-box p-4">
                    <div class="stat-title">Subdomain</div>
                    <div class="stat-value text-xl">{service.subdomain}</div>
                  </div>
                  <div class="stat bg-base-200 rounded-box p-4">
                    <div class="stat-title">Version</div>
                    <div class="stat-value text-xl">{service.version}</div>
                  </div>
                </div>

                {validations[service.id] && validations[service.id].issues.length > 0 && (
                  <div class="mt-4">
                    <div class="alert alert-warning">
                      <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                      </svg>
                      <div>
                        <h3 class="font-bold">Configuration Issues</h3>
                        <ul class="list-disc list-inside">
                          {validations[service.id].issues.map((issue, index) => (
                            <li key={index}>{issue}</li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  </div>
                )}

                <div class="card-actions justify-end mt-4">
                  <button 
                    class="btn btn-primary" 
                    onClick={() => route(`/services/${service.id}`)}
                  >
                    View Details
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}