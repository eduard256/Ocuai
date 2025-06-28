'use client';

import { useEffect, useRef, useState } from 'react';
import { X, Loader2, CheckCircle, AlertTriangle, SkipForward } from 'lucide-react';
import VideoPlayer from './VideoPlayer';

interface StreamCandidate {
  url: string;
  protocol: string;
  description: string;
  priority: number;
  working: boolean;
  error?: string;
  test_time?: string;
  latency_ms?: number;
}

interface StreamPreviewProps {
  streams: StreamCandidate[];
  currentStreamIndex: number;
  cameraID: string;
  onAccept: () => void;
  onReject: () => void;
  onNextStream: () => void;
  onClose: () => void;
}

export default function StreamPreview({ streams, currentStreamIndex, cameraID, onAccept, onReject, onNextStream, onClose }: StreamPreviewProps) {
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);
  const modalRef = useRef<HTMLDivElement>(null);
  
  // Получаем текущий поток
  const currentStream = streams[currentStreamIndex];
  const streamURL = currentStream?.url || '';
  const hasMoreStreams = currentStreamIndex < streams.length - 1;

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = 'unset';
    };
  }, [onClose]);

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div 
      className="fixed inset-0 z-[60] bg-black bg-opacity-75 flex items-center justify-center p-8"
      onClick={handleBackdropClick}
    >
      <div 
        ref={modalRef}
        className="relative w-full max-w-5xl bg-gray-900 rounded-2xl shadow-2xl"
        style={{ height: '75vh' }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="absolute top-0 left-0 right-0 z-10 bg-gradient-to-b from-gray-900 to-transparent p-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-xl font-semibold text-white">
                Stream Preview {streams.length > 1 && `(${currentStreamIndex + 1} of ${streams.length})`}
              </h3>
              {currentStream && (
                <div className="mt-1 flex items-center space-x-3">
                  <span className="text-sm text-blue-400 uppercase font-semibold">
                    {currentStream.protocol}
                  </span>
                  <span className="text-sm text-gray-400">
                    Priority: {currentStream.priority}
                  </span>
                  <span className="text-sm text-gray-400">
                    {currentStream.latency_ms || 0}ms
                  </span>
                </div>
              )}
            </div>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white transition-colors"
            >
              <X className="h-6 w-6" />
            </button>
          </div>
          
          {streamURL && (
            <p className="mt-2 text-sm text-gray-400 font-mono truncate">
              {streamURL}
            </p>
          )}
          
          {currentStream?.description && (
            <p className="mt-1 text-sm text-gray-300">
              {currentStream.description}
            </p>
          )}
        </div>

        {/* Video Container */}
        <div className="relative h-full rounded-2xl overflow-hidden bg-black">
          {isLoading && (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <Loader2 className="h-12 w-12 text-white animate-spin mx-auto mb-4" />
                <p className="text-white">Loading stream...</p>
              </div>
            </div>
          )}

          {hasError && (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <AlertTriangle className="h-12 w-12 text-yellow-500 mx-auto mb-4" />
                <p className="text-white mb-2">Failed to load stream</p>
                <p className="text-gray-400 text-sm">The stream might be incompatible or temporarily unavailable</p>
              </div>
            </div>
          )}

          {/* Video Player */}
          <VideoPlayer
            streamId={cameraID}
            cameraName="Stream Preview"
            className="w-full h-full"
          />
        </div>

        {/* Controls */}
        <div className="absolute bottom-0 left-0 right-0 z-10 bg-gradient-to-t from-gray-900 to-transparent p-6">
          <div className="flex items-center justify-center space-x-4">
            <button
              onClick={onReject}
              className="px-6 py-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg font-medium transition-colors flex items-center space-x-2"
            >
              <X className="h-5 w-5" />
              <span>Find Another Stream</span>
            </button>
            
            {hasMoreStreams && (
              <button
                onClick={onNextStream}
                className="px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors flex items-center space-x-2"
              >
                <SkipForward className="h-5 w-5" />
                <span>Next Stream</span>
              </button>
            )}
            
            <button
              onClick={onAccept}
              disabled={hasError}
              className={`px-6 py-3 rounded-lg font-medium transition-colors flex items-center space-x-2 ${
                hasError 
                  ? 'bg-gray-700 text-gray-400 cursor-not-allowed' 
                  : 'bg-green-600 hover:bg-green-700 text-white'
              }`}
            >
              <CheckCircle className="h-5 w-5" />
              <span>Use This Stream</span>
            </button>
          </div>

          <div className="mt-4 text-center">
            <p className="text-sm text-gray-400">
              Is this stream working correctly?
              {hasMoreStreams && (
                <span className="block mt-1">
                  {streams.length - currentStreamIndex - 1} more stream{streams.length - currentStreamIndex - 1 !== 1 ? 's' : ''} available
                </span>
              )}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
} 