import { AppError, getErrorAlertClass, getUserFriendlyErrorMessage, getDetailedErrorMessage } from '../types/error';

interface ErrorDisplayProps {
  error: AppError;
  onRetry?: () => void;
  showDetails?: boolean;
}

export function ErrorDisplay({ error, onRetry, showDetails = false }: ErrorDisplayProps) {
  return (
    <div class={`alert ${getErrorAlertClass(error)} mb-4`}>
      <div class="flex flex-col w-full">
        <div class="flex items-center">
          <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span class="ml-2">{showDetails ? getDetailedErrorMessage(error) : getUserFriendlyErrorMessage(error)}</span>
        </div>
        
        {error.details && showDetails && (
          <div class="mt-2 text-sm opacity-80 overflow-x-auto">
            <pre>{JSON.stringify(error.details, null, 2)}</pre>
          </div>
        )}
        
        <div class="flex justify-end mt-2">
          {onRetry && (
            <button class="btn btn-sm" onClick={onRetry}>
              Retry
            </button>
          )}
          {!showDetails && error.details && (
            <button class="btn btn-sm btn-ghost ml-2" onClick={() => window.alert(getDetailedErrorMessage(error))}>
              Details
            </button>
          )}
        </div>
      </div>
    </div>
  );
}