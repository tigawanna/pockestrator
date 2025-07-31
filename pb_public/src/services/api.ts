import PocketBase from 'pocketbase';
import { SyncOptions } from '../types/config-sync';
import { AppError, ErrorResponse } from '../types/error';

const pb = new PocketBase(window.location.origin);

// Error handling helper
const handleApiError = (error: any): never => {
  console.error('API Error:', error);
  
  // Check if it's a structured error response from our backend
  if (error?.response?.data?.error) {
    const appError: AppError = error.response.data.error;
    throw appError;
  }
  
  // Check if it's a PocketBase ClientResponseError
  if (error?.response?.message) {
    throw {
      type: 'api',
      code: error.response.code?.toString() || 'unknown',
      message: error.response.message,
      severity: 'error',
      details: error.response.data
    } as AppError;
  }
  
  // Generic error
  throw {
    type: 'unknown',
    code: 'unknown_error',
    message: error?.message || 'An unknown error occurred',
    severity: 'error'
  } as AppError;
};

export const api = {
  // Authentication
  login: async (email: string, password: string) => {
    try {
      return await pb.admins.authWithPassword(email, password);
    } catch (error) {
      return handleApiError(error);
    }
  },
  logout: () => pb.authStore.clear(),
  isAuthenticated: () => pb.authStore.isValid,
  
  // Services
  getServices: async () => {
    try {
      return await pb.collection('services').getFullList();
    } catch (error) {
      return handleApiError(error);
    }
  },
  getService: async (id: string) => {
    try {
      return await pb.collection('services').getOne(id);
    } catch (error) {
      return handleApiError(error);
    }
  },
  createService: async (data: any) => {
    try {
      return await pb.collection('services').create(data);
    } catch (error) {
      return handleApiError(error);
    }
  },
  updateService: async (id: string, data: any) => {
    try {
      return await pb.collection('services').update(id, data);
    } catch (error) {
      return handleApiError(error);
    }
  },
  deleteService: async (id: string) => {
    try {
      return await pb.collection('services').delete(id);
    } catch (error) {
      return handleApiError(error);
    }
  },
  
  // Service validation
  validateService: async (id: string) => {
    try {
      return await pb.send(`/api/services/${id}/validate`, {});
    } catch (error) {
      return handleApiError(error);
    }
  },
  
  // Service management
  getServiceLogs: async (id: string) => {
    try {
      return await pb.send(`/api/services/${id}/logs`, {});
    } catch (error) {
      return handleApiError(error);
    }
  },
  restartService: async (id: string) => {
    try {
      return await pb.send(`/api/services/${id}/restart`, {}, { method: 'POST' });
    } catch (error) {
      return handleApiError(error);
    }
  },
  syncConfig: async (id: string) => {
    try {
      return await pb.send(`/api/services/${id}/sync-config`, {}, { method: 'POST' });
    } catch (error) {
      return handleApiError(error);
    }
  },
  
  // Configuration sync
  getConfigSyncStatus: async (id: string) => {
    try {
      return await pb.send(`/api/services/${id}/config-sync-status`, {});
    } catch (error) {
      return handleApiError(error);
    }
  },
  syncConfigItem: async (id: string, options: SyncOptions) => {
    try {
      return await pb.send(`/api/services/${id}/sync-config`, options, { method: 'POST' });
    } catch (error) {
      return handleApiError(error);
    }
  },
  
  // File management
  uploadFile: async (id: string, directory: string, file: File) => {
    try {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('directory', directory);
      return await pb.send(`/api/services/${id}/upload`, formData, { method: 'POST' });
    } catch (error) {
      return handleApiError(error);
    }
  },
  getFiles: async (id: string, directory: string) => {
    try {
      return await pb.send(`/api/services/${id}/files?directory=${directory}`, {});
    } catch (error) {
      return handleApiError(error);
    }
  },
  deleteFile: async (id: string, directory: string, filename: string) => {
    try {
      return await pb.send(`/api/services/${id}/files/${filename}?directory=${directory}`, {}, { method: 'DELETE' });
    } catch (error) {
      return handleApiError(error);
    }
  },
  
  // System
  getAvailablePorts: async () => {
    try {
      return await pb.send('/api/system/ports/available', {});
    } catch (error) {
      return handleApiError(error);
    }
  },
  getPocketBaseVersions: async () => {
    try {
      return await pb.send('/api/pocketbase/versions', {});
    } catch (error) {
      return handleApiError(error);
    }
  },
  updatePocketBase: async (id: string, version: string) => {
    try {
      return await pb.send(`/api/services/${id}/update-pocketbase`, { version }, { method: 'POST' });
    } catch (error) {
      return handleApiError(error);
    }
  },
};