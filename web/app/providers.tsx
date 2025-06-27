'use client';

import { useEffect } from 'react';
import { useAuthStore } from '@/stores/auth';
import { wsClient } from '@/lib/websocket';

export function Providers({ children }: { children: React.ReactNode }) {
  const init = useAuthStore((state) => state.init);

  useEffect(() => {
    // Initialize auth state
    init();
  }, [init]);

  useEffect(() => {
    // Connect WebSocket when authenticated
    const unsubscribe = useAuthStore.subscribe((state) => {
      if (state.isAuthenticated && !state.loading) {
        wsClient.connect();
      } else {
        wsClient.disconnect();
      }
    });

    return () => {
      unsubscribe();
      wsClient.disconnect();
    };
  }, []);

  return <>{children}</>;
} 