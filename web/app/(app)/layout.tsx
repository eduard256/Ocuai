'use client';

import { useState } from 'react';
import { AuthGuard } from '@/components/auth/AuthGuard';
import { Header } from '@/components/layout/Header';
import { Sidebar } from '@/components/layout/Sidebar';

const pageTitles: { [key: string]: string } = {
  '/dashboard': 'Dashboard',
  '/cameras': 'Cameras',
  '/events': 'Events',
  '/settings': 'Settings',
};

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  // Get current page title
  const getPageTitle = () => {
    if (typeof window !== 'undefined') {
      const pathname = window.location.pathname;
      return pageTitles[pathname] || 'Ocuai';
    }
    return 'Ocuai';
  };

  return (
    <AuthGuard requireAuth={true}>
      <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
        {/* Sidebar */}
        <Sidebar 
          isOpen={sidebarOpen} 
          onClose={() => setSidebarOpen(false)} 
        />

        {/* Main content */}
        <div className="flex-1 flex flex-col min-w-0">
          {/* Header */}
          <Header 
            title={getPageTitle()}
            onToggleSidebar={() => setSidebarOpen(!sidebarOpen)}
          />

          {/* Page content */}
          <main className="flex-1 overflow-auto">
            {children}
          </main>
        </div>
      </div>
    </AuthGuard>
  );
} 