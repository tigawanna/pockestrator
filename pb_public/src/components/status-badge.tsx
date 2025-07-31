interface StatusBadgeProps {
  status: 'creating' | 'running' | 'stopped' | 'error';
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const badgeClasses = {
    creating: 'badge badge-warning',
    running: 'badge badge-success',
    stopped: 'badge badge-secondary',
    error: 'badge badge-error',
  };

  const badgeText = {
    creating: 'Creating',
    running: 'Running',
    stopped: 'Stopped',
    error: 'Error',
  };

  return (
    <span className={badgeClasses[status]}>
      {badgeText[status]}
    </span>
  );
}