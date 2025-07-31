# Pockestrator Dashboard - UI Requirements

## Project Overview

Create a React-based web dashboard for the Pockestrator PocketBase instance manager. The dashboard will provide a user-friendly interface for managing, monitoring, and validating PocketBase service configurations.

## Technical Stack

### Frontend Framework & Libraries
- **React 18** with TypeScript for type safety
- **TailwindCSS** for utility-first styling
- **shadcn/ui** component library for consistent UI components
- **TanStack Query** for server state management and data fetching
- **TanStack Router** for file-based routing
- **Zustand** for client-side global state (if needed)
- **PocketBase JS SDK** for API communication

### Build & Distribution
- **Vite** for development and build tooling
- **Embedded Assets**: Build output must be embedded in Go binary
- **Single Binary Distribution**: Follow approach from https://bindplane.com/blog/embed-react-in-golang

## Core Features & User Interface

### 1. Service Registry Dashboard
**Main View (`/services`)**
- **Service List Table/Grid**
  - Project name, version, port, status columns
  - Real-time status indicators (green/yellow/red)
  - Configuration health badges
  - Quick action buttons (start/stop/restart)
  
- **Visual Status Indicators**
  - 🟢 Healthy: All configurations match and service is running
  - 🟡 Warning: Minor configuration drift detected
  - 🔴 Error: Service down or major configuration mismatch
  - ⚪ Unknown: Unable to verify status

- **Configuration Validation Alerts**
  - Visual "squiggles" or warning icons for mismatched configs
  - Tooltip explanations for detected issues
  - One-click resolution suggestions

### 2. Service Creation Form (`/services/new`)
**Form Components:**
- **Project Name Input** (required, with uniqueness validation)
- **PocketBase Version Selector** (dropdown with latest as default)
- **Port Number Input** (auto-increment from last used, with availability check)
- **Domain Configuration** (optional subdomain prefix)
- **Advanced Options** (collapsible section)
  - Custom systemd service settings
  - Caddy proxy configurations
  - Resource limits

**Validation & Feedback:**
- Real-time form validation
- Port availability checking
- Project name uniqueness verification
- Version compatibility warnings

### 3. Service Detail View (`/services/:id`)
**Configuration Comparison Panel:**
- **Saved Configuration** (from PocketBase collection)
- **Current System Configuration** (live queries)
- **Diff Visualization** highlighting mismatches

**Configuration Sections:**
- **SystemD Service Configuration**
  - Service file location and permissions
  - ExecStart command validation
  - Port binding verification
  - Service status and logs

- **Caddy Proxy Configuration**
  - Reverse proxy rules
  - SSL certificate status
  - Domain routing validation
  - Header forwarding settings

- **PocketBase Instance Details**
  - Binary version and location
  - Database file status
  - Admin user configuration
  - API endpoint accessibility

**Action Buttons:**
- Synchronize configurations
- Restart service
- View logs
- Update instance
- Delete service (with confirmation)

### 4. System Health Dashboard (`/health`)
**Overview Metrics:**
- Total services count
- Running/stopped services ratio
- Configuration drift summary
- Resource usage overview

**System Status Checks:**
- Caddy service status
- SystemD availability
- Disk space monitoring
- Network port conflicts

## Data Flow & API Integration

### PocketBase API Communication
```typescript
// Service operations
interface ServiceRecord {
  id: string;
  project_name: string;
  port: number;
  pocketbase_version: string;
  domain: string;
  status: 'active' | 'inactive' | 'error';
  systemd_config_hash: string;
  caddy_config_hash: string;
  last_health_check: string;
}

// API endpoints
const serviceApi = {
  list: () => pb.collection('services').getList(),
  create: (data: ServiceRecord) => pb.collection('services').create(data),
  update: (id: string, data: Partial<ServiceRecord>) => pb.collection('services').update(id, data),
  delete: (id: string) => pb.collection('services').delete(id),
  validateConfig: (id: string) => pb.send('/api/services/validate', { method: 'POST', body: { id } })
};
```

### Real-time Updates
- **WebSocket Connection** for live status updates
- **Polling Strategy** for configuration validation (every 30 seconds)
- **Optimistic Updates** for immediate UI feedback

### State Management
```typescript
// Zustand store for global state
interface AppState {
  services: ServiceRecord[];
  systemHealth: SystemHealth;
  selectedService: string | null;
  isLoading: boolean;
  error: string | null;
}
```

## Component Architecture

### Layout Components
- **AppLayout**: Main application shell with navigation
- **Header**: Logo, navigation menu, user profile
- **Sidebar**: Quick navigation and service count
- **Footer**: Version info and system status

### Feature Components
- **ServiceCard**: Individual service display card
- **ServiceForm**: Create/edit service form
- **ConfigDiff**: Configuration comparison component
- **StatusBadge**: Service status indicator
- **HealthCheck**: System health monitoring widget

### UI Components (shadcn/ui)
- **Button**, **Input**, **Select**, **Textarea**
- **Dialog**, **Alert**, **Badge**, **Card**
- **Table**, **Tabs**, **Switch**, **Slider**

## Error Handling & User Experience

### Error States
- **Network Errors**: Connection timeout, API unavailable
- **Validation Errors**: Form validation, configuration conflicts
- **Permission Errors**: Insufficient system permissions
- **Service Errors**: Failed deployments, configuration issues

### Loading States
- **Skeleton Loaders** for table and card components
- **Progressive Loading** for large datasets
- **Optimistic Updates** with rollback on failure

### User Feedback
- **Toast Notifications** for success/error messages
- **Progress Indicators** for long-running operations
- **Confirmation Dialogs** for destructive actions
- **Help Tooltips** for complex configurations

## Responsive Design

### Breakpoints
- **Mobile** (320px - 768px): Stacked layouts, drawer navigation
- **Tablet** (768px - 1024px): Adaptive grid layouts
- **Desktop** (1024px+): Full feature layout with sidebars

### Mobile Optimizations
- Touch-friendly button sizes (44px minimum)
- Simplified navigation patterns
- Responsive table layouts (card view on mobile)
- Swipe gestures for common actions

## Accessibility Requirements

### WCAG 2.1 Compliance
- **Keyboard Navigation**: Full keyboard accessibility
- **Screen Reader Support**: Proper ARIA labels and roles
- **Color Contrast**: Meet AA standards for text and backgrounds
- **Focus Management**: Clear focus indicators and logical tab order

### Semantic HTML
- Proper heading hierarchy (h1 → h6)
- Form labels and descriptions
- Table headers and captions
- Button and link text clarity

## Performance Optimization

### Code Splitting
- Route-based code splitting
- Component lazy loading
- Bundle size optimization

### Caching Strategy
- API response caching with TanStack Query
- Static asset caching
- Configuration validation result caching

### Build Optimization
- Tree shaking for unused code
- CSS purging with TailwindCSS
- Asset compression and minification

This comprehensive UI specification ensures a professional, accessible, and user-friendly dashboard for managing PocketBase instances while maintaining consistency with modern web application standards.
