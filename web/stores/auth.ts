import { create } from 'zustand';
import { AuthState, User } from '@/types';
import { api } from '@/lib/api';

interface AuthStore extends AuthState {
  // Actions
  init: () => Promise<void>;
  login: (username: string, password: string) => Promise<{ success: boolean; error?: string }>;
  register: (username: string, password: string) => Promise<{ success: boolean; error?: string }>;
  logout: () => Promise<void>;
  setUser: (user: User | null) => void;
  setLoading: (loading: boolean) => void;
}

export const useAuthStore = create<AuthStore>((set, get) => ({
  // Initial state
  loading: true,
  isAuthenticated: false,
  user: null,
  setupRequired: false,

  // Actions
  init: async () => {
    try {
      set({ loading: true });

      // Check if setup is required
      const setupResponse = await api.checkSetup();
      if (setupResponse.success && setupResponse.data.setup_required) {
        set({
          loading: false,
          isAuthenticated: false,
          user: null,
          setupRequired: true,
        });
        return;
      }

      // Check auth status
      const statusResponse = await api.checkAuthStatus();
      if (statusResponse.success && statusResponse.data.authenticated) {
        set({
          loading: false,
          isAuthenticated: true,
          user: statusResponse.data.user,
          setupRequired: false,
        });
      } else {
        set({
          loading: false,
          isAuthenticated: false,
          user: null,
          setupRequired: false,
        });
      }
    } catch (error) {
      console.error('Auth init failed:', error);
      set({
        loading: false,
        isAuthenticated: false,
        user: null,
        setupRequired: false,
      });
    }
  },

  login: async (username: string, password: string) => {
    try {
      const response = await api.login(username, password);
      
      if (response.success) {
        set({
          loading: false,
          isAuthenticated: true,
          user: response.data.user,
          setupRequired: false,
        });
        return { success: true };
      }
      
      return { success: false, error: response.error || 'Login failed' };
    } catch (error: any) {
      return { success: false, error: error.message || 'Network error' };
    }
  },

  register: async (username: string, password: string) => {
    try {
      const response = await api.register(username, password);
      
      if (response.success) {
        // Check if auto-login was successful
        if (response.data.auto_login) {
          set({
            loading: false,
            isAuthenticated: true,
            user: response.data.user,
            setupRequired: false,
          });
        } else {
          set({
            loading: false,
            isAuthenticated: false,
            user: null,
            setupRequired: false,
          });
        }
        return { success: true };
      }
      
      return { success: false, error: response.error || 'Registration failed' };
    } catch (error: any) {
      return { success: false, error: error.message || 'Network error' };
    }
  },

  logout: async () => {
    try {
      await api.logout();
    } catch (error) {
      console.error('Logout error:', error);
    }
    
    set({
      loading: false,
      isAuthenticated: false,
      user: null,
      setupRequired: false,
    });
  },

  setUser: (user) => set({ user }),
  setLoading: (loading) => set({ loading }),
})); 