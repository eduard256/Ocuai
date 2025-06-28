'use client';

import { useEffect } from 'react';
import { Edit, Trash2, RefreshCw } from 'lucide-react';
import { useAppStore } from '@/stores/app';
import VideoPlayer from '@/components/cameras/VideoPlayer';

export default function CamerasPage() {
  const { cameras, loadCameras, deleteCamera } = useAppStore();

  useEffect(() => {
    loadCameras();
  }, [loadCameras]);

  const handleDelete = async (id: string) => {
    if (confirm('Are you sure you want to delete this camera?')) {
      await deleteCamera(id);
    }
  };

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Cameras</h1>
      </div>

      {/* Cameras Grid */}
      {cameras.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {cameras.map((camera) => (
            <div key={camera.id} className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
              {/* Camera Preview */}
              <VideoPlayer 
                streamId={camera.id} 
                cameraName={camera.name}
                className="aspect-video"
              />

              {/* Camera Info */}
              <div className="p-4">
                <h3 className="font-medium text-gray-900 dark:text-white mb-1">{camera.name}</h3>
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">ID: {camera.id}</p>
                
                {/* Features */}
                <div className="flex flex-wrap gap-1 mb-3">
                  {camera.motion_detection && (
                    <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/20 text-blue-800 dark:text-blue-300 text-xs rounded">
                      Motion
                    </span>
                  )}
                  {camera.ai_detection && (
                    <span className="px-2 py-1 bg-purple-100 dark:bg-purple-900/20 text-purple-800 dark:text-purple-300 text-xs rounded">
                      AI
                    </span>
                  )}
                </div>

                {/* Actions */}
                <div className="flex items-center justify-between">
                  <button className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                    <RefreshCw className="h-4 w-4" />
                  </button>
                  <div className="flex items-center space-x-1">
                    <button className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                      <Edit className="h-4 w-4" />
                    </button>
                    <button 
                      onClick={() => handleDelete(camera.id)}
                      className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="text-center py-12">
          <div className="mx-auto h-24 w-24 text-gray-400">
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 002 2v8a2 2 0 002 2z" />
            </svg>
          </div>
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">No cameras</h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Cameras will appear here when available.</p>
        </div>
      )}
    </div>
  );
} 