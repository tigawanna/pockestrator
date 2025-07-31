import { ServiceValidation } from '../types/service';

interface ValidationIndicatorProps {
  validation: ServiceValidation | null;
  loading?: boolean;
}

export function ValidationIndicator({ validation, loading = false }: ValidationIndicatorProps) {
  if (loading) {
    return <span className="loading loading-spinner loading-sm"></span>;
  }

  if (!validation) {
    return <span className="badge badge-outline">Not validated</span>;
  }

  const allValid = 
    validation.systemd_exists && 
    validation.systemd_running && 
    validation.caddy_configured && 
    validation.binary_exists && 
    validation.port_matches;

  if (allValid) {
    return (
      <div className="flex items-center">
        <span className="badge badge-success gap-1">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" className="inline-block w-4 h-4 stroke-current">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7"></path>
          </svg>
          Valid
        </span>
      </div>
    );
  }

  return (
    <div className="flex items-center">
      <span className="badge badge-warning gap-1">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" className="inline-block w-4 h-4 stroke-current">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
        </svg>
        Issues: {validation.issues.length}
      </span>
    </div>
  );
}