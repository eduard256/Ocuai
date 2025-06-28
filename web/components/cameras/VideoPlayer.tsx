'use client';

import { useEffect, useRef, useState } from 'react';
import { Play, Pause, Volume2, VolumeX, Maximize, Loader2, AlertCircle } from 'lucide-react';

interface VideoPlayerProps {
  streamId: string;
  cameraName: string;
  className?: string;
}

export default function VideoPlayer({ streamId, cameraName, className = '' }: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pc, setPc] = useState<RTCPeerConnection | null>(null);

  // go2rtc WebRTC URL - используем проксированную ссылку через Next.js
  const webrtcUrl = `/go2rtc/api/webrtc?src=${streamId}`;

  useEffect(() => {
    return () => {
      // Cleanup WebRTC connection on unmount
      if (pc) {
        pc.close();
      }
    };
  }, [pc]);

  const startWebRTC = async () => {
    if (!videoRef.current) return;

    setIsLoading(true);
    setError(null);

    try {
      console.log('Starting WebRTC connection to:', webrtcUrl);
      
      // Сначала проверим доступность go2rtc
      const healthCheck = await fetch('/go2rtc/api/streams', {
        method: 'GET',
      }).catch(err => {
        console.error('go2rtc health check failed:', err);
        throw new Error('go2rtc server is not available');
      });

      if (!healthCheck.ok) {
        throw new Error('go2rtc server is not responding');
      }

      // Create peer connection
      const peerConnection = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      // Handle connection state changes
      peerConnection.onconnectionstatechange = () => {
        console.log('WebRTC connection state:', peerConnection.connectionState);
        if (peerConnection.connectionState === 'failed') {
          setError('WebRTC connection failed');
          setIsLoading(false);
        }
      };

      // Set up video element
      peerConnection.ontrack = (event) => {
        console.log('Received track:', event.track.kind);
        if (videoRef.current && event.streams[0]) {
          videoRef.current.srcObject = event.streams[0];
          setIsLoading(false);
          setIsPlaying(true);
        }
      };

      // Add transceivers for receiving video/audio
      peerConnection.addTransceiver('video', { direction: 'recvonly' });
      peerConnection.addTransceiver('audio', { direction: 'recvonly' });

      // Create offer
      const offer = await peerConnection.createOffer();
      await peerConnection.setLocalDescription(offer);

      console.log('Sending WebRTC offer to go2rtc...');

      // Send offer to go2rtc and get answer
      const response = await fetch(webrtcUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(offer)
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('go2rtc response error:', response.status, errorText);
        throw new Error(`go2rtc error: ${response.status} - ${errorText}`);
      }

      const answer = await response.json();
      console.log('Received WebRTC answer from go2rtc');
      
      await peerConnection.setRemoteDescription(answer);

      setPc(peerConnection);
    } catch (err) {
      console.error('WebRTC error:', err);
      console.log('Attempting fallback to HLS...');
      
      // Пробуем HLS как fallback
      try {
        useHLS();
      } catch (hlsErr) {
        console.error('HLS fallback failed:', hlsErr);
        const errorMessage = err instanceof Error ? err.message : 'Unknown error';
        setError(`Failed to connect via WebRTC and HLS: ${errorMessage}`);
        setIsLoading(false);
      }
    }
  };

  const stopWebRTC = () => {
    if (pc) {
      pc.close();
      setPc(null);
    }
    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
    setIsPlaying(false);
  };

  const handlePlayPause = () => {
    if (isPlaying) {
      stopWebRTC();
    } else {
      startWebRTC();
    }
  };

  const handleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !isMuted;
      setIsMuted(!isMuted);
    }
  };

  const handleFullscreen = () => {
    if (containerRef.current) {
      if (!document.fullscreenElement) {
        containerRef.current.requestFullscreen();
      } else {
        document.exitFullscreen();
      }
    }
  };

  // Alternative: Use MJPEG stream for simpler implementation
  const useMJPEG = () => {
    if (!videoRef.current) return;
    
    const mjpegUrl = `/go2rtc/api/frame.mjpeg?src=${streamId}`;
    
    console.log('Switching to MJPEG stream:', mjpegUrl);
    
    // Use MJPEG as img src for better compatibility
    const img = new Image();
    img.onload = () => {
      if (videoRef.current) {
        // Create a canvas to display MJPEG
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        
        // Set canvas size to match video element
        canvas.width = videoRef.current.clientWidth || 640;
        canvas.height = videoRef.current.clientHeight || 480;
        
        // Replace video with canvas
        const parent = videoRef.current.parentNode;
        if (parent) {
          parent.replaceChild(canvas, videoRef.current);
        }
        
        setIsPlaying(true);
        setIsLoading(false);
      }
    };
    
    img.onerror = () => {
      setError('MJPEG stream not available');
      setIsLoading(false);
    };
    
    img.src = mjpegUrl;
  };

  // Alternative: Use HLS stream
  const useHLS = () => {
    if (!videoRef.current) return;
    
    const hlsUrl = `/go2rtc/api/stream.m3u8?src=${streamId}`;
    
    console.log('Switching to HLS stream:', hlsUrl);
    
    // For browsers that support HLS natively (Safari)
    if (videoRef.current.canPlayType('application/vnd.apple.mpegurl')) {
      videoRef.current.src = hlsUrl;
      setIsPlaying(true);
      setIsLoading(false);
    } else {
      // Fallback to MJPEG if HLS not supported
      useMJPEG();
    }
  };

  return (
    <div ref={containerRef} className={`relative bg-black rounded-lg overflow-hidden ${className}`}>
      <video
        ref={videoRef}
        className="w-full h-full object-contain"
        autoPlay
        playsInline
        muted={isMuted}
      />

      {/* Loading overlay */}
      {isLoading && (
        <div className="absolute inset-0 flex items-center justify-center bg-black bg-opacity-50">
          <Loader2 className="h-8 w-8 text-white animate-spin" />
        </div>
      )}

      {/* Error overlay */}
      {error && (
        <div className="absolute inset-0 flex items-center justify-center bg-black bg-opacity-75">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-2" />
            <p className="text-white text-sm mb-4">{error}</p>
            <div className="flex flex-col gap-2">
              <button
                onClick={() => {
                  setError(null);
                  useHLS();
                }}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
              >
                Try HLS Stream
              </button>
              <button
                onClick={() => {
                  setError(null);
                  useMJPEG();
                }}
                className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 transition-colors"
              >
                Try MJPEG Stream
              </button>
              <button
                onClick={() => {
                  setError(null);
                  startWebRTC();
                }}
                className="px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700 transition-colors"
              >
                Retry WebRTC
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Play button overlay when not playing */}
      {!isPlaying && !isLoading && !error && (
        <div className="absolute inset-0 flex items-center justify-center bg-black bg-opacity-50 cursor-pointer" onClick={handlePlayPause}>
          <Play className="h-16 w-16 text-white" />
        </div>
      )}

      {/* Controls */}
      <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black to-transparent p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <button
              onClick={handlePlayPause}
              className="p-2 text-white hover:bg-white/20 rounded transition-colors"
            >
              {isPlaying ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
            </button>
            <button
              onClick={handleMute}
              className="p-2 text-white hover:bg-white/20 rounded transition-colors"
            >
              {isMuted ? <VolumeX className="h-5 w-5" /> : <Volume2 className="h-5 w-5" />}
            </button>
            <span className="text-white text-sm ml-2">{cameraName}</span>
          </div>
          <button
            onClick={handleFullscreen}
            className="p-2 text-white hover:bg-white/20 rounded transition-colors"
          >
            <Maximize className="h-5 w-5" />
          </button>
        </div>
      </div>
    </div>
  );
} 