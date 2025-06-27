'use client';

import { useEffect } from 'react';
import { Calendar, Clock, Camera, AlertCircle } from 'lucide-react';
import { useAppStore } from '@/stores/app';

export default function EventsPage() {
  const { events, loadEvents } = useAppStore();

  useEffect(() => {
    loadEvents({ limit: 50 });
  }, [loadEvents]);

  const getEventIcon = (type: string) => {
    if (type === 'motion') {
      return <AlertCircle className="h-5 w-5 text-yellow-500" />;
    }
    return <Camera className="h-5 w-5 text-blue-500" />;
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', { 
      year: 'numeric', 
      month: 'short', 
      day: 'numeric' 
    });
  };

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleTimeString('en-US', { 
      hour: '2-digit', 
      minute: '2-digit' 
    });
  };

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">Events</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Recent activity from all cameras
        </p>
      </div>

      {/* Events List */}
      {events.length > 0 ? (
        <div className="bg-white dark:bg-gray-800 shadow overflow-hidden sm:rounded-md">
          <ul className="divide-y divide-gray-200 dark:divide-gray-700">
            {events.map((event) => (
              <li key={event.id}>
                <div className="px-4 py-4 sm:px-6 hover:bg-gray-50 dark:hover:bg-gray-700">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <div className="flex-shrink-0">
                        {getEventIcon(event.type)}
                      </div>
                      <div className="ml-4">
                        <div className="text-sm font-medium text-gray-900 dark:text-white">
                          {event.camera_name}
                        </div>
                        <div className="text-sm text-gray-500 dark:text-gray-400">
                          {event.type === 'motion' 
                            ? 'Motion detected' 
                            : `${event.object_class} detected (${Math.round((event.confidence || 0) * 100)}%)`
                          }
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center text-sm text-gray-500 dark:text-gray-400">
                      <Calendar className="flex-shrink-0 mr-1.5 h-4 w-4" />
                      <span>{formatDate(event.created_at)}</span>
                      <Clock className="flex-shrink-0 ml-4 mr-1.5 h-4 w-4" />
                      <span>{formatTime(event.created_at)}</span>
                    </div>
                  </div>
                  {event.description && (
                    <div className="mt-2 text-sm text-gray-600 dark:text-gray-300">
                      {event.description}
                    </div>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <div className="text-center py-12">
          <AlertCircle className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">No events</h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Events will appear here when motion or objects are detected
          </p>
        </div>
      )}
    </div>
  );
} 