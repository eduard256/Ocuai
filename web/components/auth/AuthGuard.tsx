'use client';

import { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore } from '@/stores/auth';

interface AuthGuardProps {
  children: React.ReactNode;
  requireAuth?: boolean;
}

export function AuthGuard({ children, requireAuth = true }: AuthGuardProps) {
  const router = useRouter();
  const pathname = usePathname();
  const { loading, isAuthenticated, setupRequired } = useAuthStore();

  useEffect(() => {
    if (!loading) {
      if (requireAuth) {
        // For protected routes
        if (setupRequired) {
          router.replace('/register');
          return;
        }
        if (!isAuthenticated) {
          router.replace('/login');
          return;
        }
      } else {
        // For auth pages (login/register)
        if (isAuthenticated) {
          router.replace('/dashboard');
          return;
        }
        
        // Block register page if setup is not required (users exist)
        if (pathname === '/register' && !setupRequired) {
          router.replace('/login');
          return;
        }
        
        // Redirect to register if setup is required and not on register page
        if (setupRequired && pathname !== '/register') {
          router.replace('/register');
          return;
        }
      }
    }
  }, [loading, isAuthenticated, setupRequired, requireAuth, router, pathname]);

  // Show loading state
  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <div className="relative">
            <div className="animate-spin rounded-full h-16 w-16 border-4 border-gray-200 border-t-indigo-600 mx-auto mb-4"></div>
            <div className="absolute inset-0 rounded-full h-16 w-16 border-4 border-transparent border-t-indigo-400 mx-auto animate-ping opacity-20"></div>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-lg font-medium">Loading Ocuai...</p>
          <div className="mt-2 flex items-center justify-center space-x-1">
            <div className="w-2 h-2 bg-indigo-600 rounded-full animate-bounce" style={{animationDelay: '0ms'}}></div>
            <div className="w-2 h-2 bg-indigo-600 rounded-full animate-bounce" style={{animationDelay: '150ms'}}></div>
            <div className="w-2 h-2 bg-indigo-600 rounded-full animate-bounce" style={{animationDelay: '300ms'}}></div>
          </div>
        </div>
      </div>
    );
  }

  // Prevent flash of wrong content during redirects
  if (requireAuth && (!isAuthenticated || setupRequired)) {
    return null;
  }

  if (!requireAuth && isAuthenticated && !setupRequired) {
    return null;
  }

  return <>{children}</>;
} 