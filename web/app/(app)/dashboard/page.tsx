'use client';

import { useEffect } from 'react';
import { Camera, Activity, AlertTriangle, Clock } from 'lucide-react';
import { useAppStore } from '@/stores/app';

export default function DashboardPage() {
  const { 
    systemStats, 
    cameras, 
    events,
    currentTime,
    loadCameras,
    loadEvents,
    loadStats
  } = useAppStore();

  useEffect(() => {
    // Load initial data
    loadCameras();
    loadEvents({ limit: 5 });
    loadStats();

    // Refresh data every 30 seconds
    const interval = setInterval(() => {
      loadStats();
    }, 30000);

    return () => clearInterval(interval);
  }, [loadCameras, loadEvents, loadStats]);

  const formatUptime = (seconds: number) => {
    if (!seconds) return '0m';
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  };

  const onlineCameras = cameras.filter(cam => cam.status === 'online');
  const recentEvents = events.slice(0, 5);

  return (
    <div className="p-6 space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
          Dashboard
        </h2>
        <div className="text-sm text-gray-500 dark:text-gray-400">
          Last update: {currentTime}
        </div>
      </div>

      {/* Live Data Indicator */}
      <div className="flex items-center space-x-2 text-sm text-gray-500">
        <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
        <span>Live data updates every 5 seconds</span>
        <span className="text-xs">({currentTime})</span>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Total Cameras */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Camera className="h-8 w-8 text-blue-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Cameras</p>
              <p className="text-2xl font-semibold text-gray-900 dark:text-white">
                {systemStats.cameras_total || 0}
              </p>
              <p className="text-xs text-green-600">
                {systemStats.cameras_online || 0} online
              </p>
            </div>
          </div>
        </div>

        {/* Today's Events */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Activity className="h-8 w-8 text-yellow-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Events Today</p>
              <p className="text-2xl font-semibold text-gray-900 dark:text-white">
                {systemStats.events_today || 0}
              </p>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                Last 24 hours
              </p>
            </div>
          </div>
        </div>

        {/* Total Events */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <AlertTriangle className="h-8 w-8 text-red-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Total Events</p>
              <p className="text-2xl font-semibold text-gray-900 dark:text-white">
                {systemStats.events_total || 0}
              </p>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                All time
              </p>
            </div>
          </div>
        </div>

        {/* System Uptime */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Clock className="h-8 w-8 text-green-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Uptime</p>
              <p className="text-2xl font-semibold text-gray-900 dark:text-white">
                {formatUptime(systemStats.system_uptime)}
              </p>
              <p className="text-xs text-green-600">
                System stable
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Two Column Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Active Cameras */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                Active Cameras
              </h3>
              <span className="text-sm text-gray-500 dark:text-gray-400">
                {onlineCameras.length} of {cameras.length}
              </span>
            </div>
          </div>
          <div className="p-6">
            {onlineCameras.length > 0 ? (
              <div className="space-y-4">
                {onlineCameras.slice(0, 4).map((camera) => (
                  <div key={camera.id} className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                      <div>
                        <p className="text-sm font-medium text-gray-900 dark:text-white">{camera.name}</p>
                        <p className="text-xs text-gray-500 dark:text-gray-400">ID: {camera.id}</p>
                      </div>
                    </div>
                    <span className="text-xs text-green-600 dark:text-green-400">Online</span>
                  </div>
                ))}
                {cameras.length > 4 && (
                  <a href="/cameras" className="block text-center text-sm text-blue-600 hover:text-blue-700">
                    View all cameras ({cameras.length})
                  </a>
                )}
              </div>
            ) : (
              <div className="text-center py-8">
                <Camera className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
                  No active cameras
                </h3>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Add a camera to start monitoring
                </p>
                <div className="mt-6">
                  <a
                    href="/cameras"
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
                  >
                    <Camera className="-ml-1 mr-2 h-5 w-5" />
                    Add Camera
                  </a>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Recent Events */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                Recent Events
              </h3>
              <span className="text-sm text-gray-500 dark:text-gray-400">
                Latest 5
              </span>
            </div>
          </div>
          <div className="p-6">
            {recentEvents.length > 0 ? (
              <div className="space-y-4">
                {recentEvents.map((event) => (
                  <div key={event.id} className="flex items-start space-x-3 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                    <div className={`flex-shrink-0 w-2 h-2 rounded-full mt-1.5 ${
                      event.type === 'motion' ? 'bg-yellow-500' : 'bg-blue-500'
                    }`} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-900 dark:text-white">
                        {event.camera_name}
                      </p>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        {event.type === 'motion' ? 'Motion detected' : `AI: ${event.object_class}`}
                      </p>
                      <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                        {new Date(event.created_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                ))}
                <a href="/events" className="block text-center text-sm text-blue-600 hover:text-blue-700">
                  View all events
                </a>
              </div>
            ) : (
              <div className="text-center py-8">
                <Activity className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
                  No events
                </h3>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Events will appear here when detected
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
} 