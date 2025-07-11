import axios, { AxiosInstance, AxiosError } from 'axios';

interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

class ApiClient {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: '/api',
      timeout: 120000,
      withCredentials: true,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        // We're using cookie-based auth, so no need to add tokens
        return config;
      },
      (error) => {
        return Promise.reject(error);
      }
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => {
        return response.data;
      },
      (error: AxiosError<ApiResponse>) => {
        if (error.response?.status === 401) {
          // Handle unauthorized - redirect to login
          if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
            window.location.href = '/login';
          }
        }
        
        const errorMessage = error.response?.data?.error || error.message || 'An error occurred';
        return Promise.reject(new Error(errorMessage));
      }
    );
  }

  // Auth endpoints
  async checkSetup(): Promise<ApiResponse<{ setup_required: boolean; has_users: boolean }>> {
    return this.client.get('/auth/setup');
  }

  async register(username: string, password: string): Promise<ApiResponse<{ user: any; auto_login: boolean; message: string }>> {
    return this.client.post('/auth/register', { username, password });
  }

  async login(username: string, password: string): Promise<ApiResponse<{ user: any; message: string }>> {
    return this.client.post('/auth/login', { username, password });
  }

  async logout(): Promise<ApiResponse> {
    return this.client.post('/auth/logout');
  }

  async checkAuthStatus(): Promise<ApiResponse<{ authenticated: boolean; user?: any }>> {
    return this.client.get('/auth/status');
  }

  // Camera endpoints
  async getCameras(): Promise<ApiResponse<any[]>> {
    return this.client.get('/cameras');
  }

  async getCamera(id: string): Promise<ApiResponse> {
    return this.client.get(`/cameras/${id}`);
  }

  async updateCamera(id: string, data: any): Promise<ApiResponse> {
    return this.client.put(`/cameras/${id}`, data);
  }

  async deleteCamera(id: string): Promise<ApiResponse> {
    return this.client.delete(`/cameras/${id}`);
  }

  async testCamera(id: string): Promise<ApiResponse> {
    return this.client.post(`/cameras/${id}/test`);
  }

  // Event endpoints
  async getEvents(params?: { limit?: number; offset?: number; camera_id?: string }): Promise<ApiResponse<any[]>> {
    return this.client.get('/events', { params });
  }

  async getEvent(id: number): Promise<ApiResponse> {
    return this.client.get(`/events/${id}`);
  }

  async deleteEvent(id: number): Promise<ApiResponse> {
    return this.client.delete(`/events/${id}`);
  }

  // System endpoints
  async getStats(): Promise<ApiResponse<any>> {
    return this.client.get('/stats');
  }

  async getHealth(): Promise<ApiResponse> {
    return this.client.get('/health');
  }

  // Settings endpoints
  async getSettings(): Promise<ApiResponse> {
    return this.client.get('/settings');
  }

  async updateSettings(settings: any): Promise<ApiResponse> {
    return this.client.put('/settings', settings);
  }
}

export const api = new ApiClient();

// Export convenience methods
export const API = {
  // Auth
  checkSetup: () => api.checkSetup(),
  register: (username: string, password: string) => api.register(username, password),
  login: (username: string, password: string) => api.login(username, password),
  logout: () => api.logout(),
  checkAuthStatus: () => api.checkAuthStatus(),
  
  // Cameras
  getCameras: () => api.getCameras(),
  getCamera: (id: string) => api.getCamera(id),
  updateCamera: (id: string, data: any) => api.updateCamera(id, data),
  deleteCamera: (id: string) => api.deleteCamera(id),
  testCamera: (id: string) => api.testCamera(id),
  
  // Events
  getEvents: (params?: { limit?: number; offset?: number; camera_id?: string }) => api.getEvents(params),
  getEvent: (id: number) => api.getEvent(id),
  deleteEvent: (id: number) => api.deleteEvent(id),
  
  // System
  getStats: () => api.getStats(),
  getHealth: () => api.getHealth(),
  
  // Settings
  getSettings: () => api.getSettings(),
  updateSettings: (settings: any) => api.updateSettings(settings),
}; 