export interface User {
  id: number;
  username: string;
  role: string;
}

export interface AuthState {
  loading: boolean;
  isAuthenticated: boolean;
  user: User | null;
  setupRequired: boolean;
}

export interface Camera {
  id: string;
  name: string;
  rtsp_url: string;
  status: 'online' | 'offline' | 'error';
  motion_detection: boolean;
  ai_detection: boolean;
  created_at: string;
  updated_at: string;
  last_seen?: string;
}

export interface Event {
  id: number;
  camera_id: string;
  camera_name: string;
  type: 'motion' | 'ai_detection';
  description: string;
  confidence?: number;
  object_class?: string;
  snapshot_url?: string;
  video_url?: string;
  created_at: string;
}

export interface SystemStats {
  cameras_total: number;
  cameras_online: number;
  events_total: number;
  events_today: number;
  system_uptime: number;
  timestamp?: number;
  current_time?: string;
}

export interface Notification {
  id: number;
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
  timestamp: number;
}

export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface Settings {
  ai_enabled: boolean;
  ai_threshold: number;
  motion_detection: boolean;
  telegram_enabled: boolean;
  notification_hours: string[];
  retention_days: number;
  max_video_size_mb: number;
}

export interface WebSocketMessage {
  type: 'stats_update' | 'camera_status' | 'new_event' | 'motion_detected' | 
        'ai_detection' | 'system_alert' | 'camera_added' | 'camera_removed' | 'camera_updated';
  data?: any;
  camera_id?: string;
  camera_name?: string;
  status?: string;
  event?: Event;
  camera?: Camera;
  message?: string;
  level?: string;
  object_class?: string;
  confidence?: number;
} 