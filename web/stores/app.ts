import { create } from 'zustand';
import { Camera, Event, SystemStats, Notification, WebSocketMessage } from '@/types';
import { api } from '@/lib/api';
import { wsClient } from '@/lib/websocket';

interface AppStore {
  // State
  cameras: Camera[];
  events: Event[];
  systemStats: SystemStats;
  notifications: Notification[];
  currentTime: string;
  
  // Actions
  loadCameras: () => Promise<void>;
  loadEvents: (params?: { limit?: number; offset?: number; camera_id?: string }) => Promise<void>;
  loadStats: () => Promise<void>;
  addNotification: (message: string, type: Notification['type'], duration?: number) => void;
  removeNotification: (id: number) => void;
  updateCurrentTime: () => void;
  
  // WebSocket handlers
  handleWebSocketMessage: (message: WebSocketMessage) => void;
  updateCameraStatus: (cameraId: string, status: Camera['status']) => void;
  addEvent: (event: Event) => void;
  
  // CRUD operations
  createCamera: (data: any) => Promise<{ success: boolean; error?: string }>;
  updateCamera: (id: string, data: any) => Promise<{ success: boolean; error?: string }>;
  deleteCamera: (id: string) => Promise<{ success: boolean; error?: string }>;
}

let notificationId = 0;

export const useAppStore = create<AppStore>((set, get) => ({
  // Initial state
  cameras: [],
  events: [],
  systemStats: {
    cameras_total: 0,
    cameras_online: 0,
    events_total: 0,
    events_today: 0,
    system_uptime: 0,
  },
  notifications: [],
  currentTime: new Date().toLocaleTimeString('ru-RU'),

  // Actions
  loadCameras: async () => {
    try {
      const response = await api.getCameras();
      if (response.success) {
        set({ cameras: response.data || [] });
      }
    } catch (error) {
      console.error('Failed to load cameras:', error);
    }
  },

  loadEvents: async (params) => {
    try {
      const response = await api.getEvents(params);
      if (response.success) {
        set({ events: response.data || [] });
      }
    } catch (error) {
      console.error('Failed to load events:', error);
    }
  },

  loadStats: async () => {
    try {
      const response = await api.getStats();
      if (response.success) {
        set({ systemStats: response.data || get().systemStats });
      }
    } catch (error) {
      console.error('Failed to load stats:', error);
    }
  },

  addNotification: (message, type, duration = 5000) => {
    const id = ++notificationId;
    const notification: Notification = { id, message, type, timestamp: Date.now() };
    
    set(state => ({ notifications: [...state.notifications, notification] }));
    
    if (duration > 0) {
      setTimeout(() => {
        get().removeNotification(id);
      }, duration);
    }
  },

  removeNotification: (id) => {
    set(state => ({
      notifications: state.notifications.filter(n => n.id !== id)
    }));
  },

  updateCurrentTime: () => {
    set({ currentTime: new Date().toLocaleTimeString('ru-RU') });
  },

  // WebSocket handlers
  handleWebSocketMessage: (message) => {
    const { addNotification, updateCameraStatus, addEvent } = get();
    
    switch (message.type) {
      case 'stats_update':
        if (message.data) {
          set({ systemStats: message.data });
        }
        break;

      case 'camera_status':
        if (message.camera_id && message.status) {
          updateCameraStatus(message.camera_id, message.status as Camera['status']);
          if (message.status === 'offline') {
            addNotification(`Camera "${message.camera_name}" disconnected`, 'warning', 8000);
          } else if (message.status === 'online') {
            addNotification(`Camera "${message.camera_name}" connected`, 'success', 5000);
          }
        }
        break;

      case 'new_event':
        if (message.event) {
          addEvent(message.event);
          const eventType = message.event.type === 'motion' ? 'Motion' : 'Object';
          addNotification(
            `${message.event.camera_name}: ${eventType} detected`,
            'info',
            6000
          );
        }
        break;

      case 'motion_detected':
        addNotification(
          `${message.camera_name}: Motion detected`,
          'warning',
          5000
        );
        break;

      case 'ai_detection':
        addNotification(
          `${message.camera_name}: Detected "${message.object_class}" (${Math.round((message.confidence || 0) * 100)}%)`,
          'info',
          7000
        );
        break;

      case 'system_alert':
        addNotification(
          message.message || 'System alert',
          (message.level as any) || 'warning',
          10000
        );
        break;

      case 'camera_added':
        if (message.camera) {
          set(state => ({ cameras: [...state.cameras, message.camera!] }));
          addNotification(`Added new camera: ${message.camera.name}`, 'success', 5000);
        }
        break;

      case 'camera_removed':
        if (message.camera_id) {
          set(state => ({
            cameras: state.cameras.filter(c => c.id !== message.camera_id)
          }));
          addNotification(`Camera removed: ${message.camera_name}`, 'info', 5000);
        }
        break;

      case 'camera_updated':
        if (message.camera) {
          set(state => ({
            cameras: state.cameras.map(c =>
              c.id === message.camera!.id ? { ...c, ...message.camera } : c
            )
          }));
        }
        break;
    }
  },

  updateCameraStatus: (cameraId, status) => {
    set(state => ({
      cameras: state.cameras.map(camera =>
        camera.id === cameraId
          ? { ...camera, status, last_seen: new Date().toISOString() }
          : camera
      )
    }));
    
    // Update stats
    const cameras = get().cameras;
    const onlineCount = cameras.filter(c => c.status === 'online').length;
    set(state => ({
      systemStats: {
        ...state.systemStats,
        cameras_total: cameras.length,
        cameras_online: onlineCount,
      }
    }));
  },

  addEvent: (event) => {
    set(state => {
      const newEvents = [event, ...state.events].slice(0, 100); // Keep last 100 events
      return { events: newEvents };
    });
  },

  // CRUD operations
  createCamera: async (data) => {
    try {
      const response = await api.createCamera(data);
      if (response.success) {
        await get().loadCameras();
        get().addNotification('Camera added successfully', 'success');
        return { success: true };
      }
      return { success: false, error: response.error };
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  },

  updateCamera: async (id, data) => {
    try {
      const response = await api.updateCamera(id, data);
      if (response.success) {
        await get().loadCameras();
        get().addNotification('Camera updated successfully', 'success');
        return { success: true };
      }
      return { success: false, error: response.error };
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  },

  deleteCamera: async (id) => {
    try {
      const response = await api.deleteCamera(id);
      if (response.success) {
        await get().loadCameras();
        get().addNotification('Camera deleted successfully', 'success');
        return { success: true };
      }
      return { success: false, error: response.error };
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  },
}));

// Initialize WebSocket subscription
if (typeof window !== 'undefined') {
  wsClient.subscribe((message) => {
    useAppStore.getState().handleWebSocketMessage(message);
  });
  
  // Update time every second
  setInterval(() => {
    useAppStore.getState().updateCurrentTime();
  }, 1000);
} 