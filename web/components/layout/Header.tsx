'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Menu, Bell, User, LogOut, Wifi, WifiOff } from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import { useAppStore } from '@/stores/app';

interface HeaderProps {
  title: string;
  onToggleSidebar: () => void;
}

export function Header({ title, onToggleSidebar }: HeaderProps) {
  const router = useRouter();
  const { user, logout } = useAuthStore();
  const { notifications, currentTime, removeNotification } = useAppStore();
  const [showNotifications, setShowNotifications] = useState(false);
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [isOnline, setIsOnline] = useState(true);

  useEffect(() => {
    const checkConnection = () => setIsOnline(navigator.onLine);
    
    window.addEventListener('online', checkConnection);
    window.addEventListener('offline', checkConnection);
    
    return () => {
      window.removeEventListener('online', checkConnection);
      window.removeEventListener('offline', checkConnection);
    };
  }, []);

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  return (
    <header className="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700">
      <div className="flex items-center justify-between px-6 py-4">
        {/* Left side */}
        <div className="flex items-center space-x-4">
          {/* Mobile menu button */}
          <button
            onClick={onToggleSidebar}
            className="p-2 rounded-md text-gray-500 hover:text-gray-900 hover:bg-gray-100 
                     dark:text-gray-400 dark:hover:text-white dark:hover:bg-gray-700 
                     focus:outline-none focus:ring-2 focus:ring-indigo-500 lg:hidden"
          >
            <Menu size={20} />
          </button>

          {/* Title */}
          <h1 className="text-xl font-semibold text-gray-900 dark:text-white">
            {title}
          </h1>

          {/* Status indicators */}
          <div className="hidden sm:flex items-center space-x-3">
            {/* Connection status */}
            <div className="flex items-center space-x-1">
              {isOnline ? (
                <>
                  <Wifi size={16} className="text-green-500" />
                  <span className="text-sm text-green-600 dark:text-green-400">Online</span>
                </>
              ) : (
                <>
                  <WifiOff size={16} className="text-red-500" />
                  <span className="text-sm text-red-600 dark:text-red-400">Offline</span>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Right side */}
        <div className="flex items-center space-x-3">
          {/* Notifications */}
          <div className="relative">
            <button
              onClick={() => setShowNotifications(!showNotifications)}
              className="p-2 rounded-full text-gray-500 hover:text-gray-900 hover:bg-gray-100 
                       dark:text-gray-400 dark:hover:text-white dark:hover:bg-gray-700 
                       focus:outline-none focus:ring-2 focus:ring-indigo-500 relative"
            >
              <Bell size={20} />
              {notifications.length > 0 && (
                <span className="absolute -top-1 -right-1 inline-flex items-center justify-center 
                             px-2 py-1 text-xs font-bold leading-none text-white 
                             bg-red-500 rounded-full">
                  {notifications.length}
                </span>
              )}
            </button>

            {/* Notifications dropdown */}
            {showNotifications && (
              <>
                <div className="absolute right-0 mt-2 w-80 bg-white dark:bg-gray-800 
                            rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 
                            z-50 max-h-96 overflow-auto">
                  {notifications.length > 0 ? (
                    <>
                      <div className="p-3 border-b border-gray-200 dark:border-gray-700">
                        <h3 className="text-sm font-medium text-gray-900 dark:text-white">
                          Notifications
                        </h3>
                      </div>
                      <div className="divide-y divide-gray-200 dark:divide-gray-700">
                        {notifications.map((notification) => (
                          <div key={notification.id} className="p-3 hover:bg-gray-50 dark:hover:bg-gray-700">
                            <div className="flex items-start space-x-3">
                              <div className="flex-shrink-0">
                                <div className={`w-2 h-2 rounded-full mt-2 ${
                                  notification.type === 'error' ? 'bg-red-500' :
                                  notification.type === 'warning' ? 'bg-yellow-500' :
                                  notification.type === 'success' ? 'bg-green-500' :
                                  'bg-blue-500'
                                }`} />
                              </div>
                              <div className="flex-1 min-w-0">
                                <p className="text-sm text-gray-900 dark:text-white">
                                  {notification.message}
                                </p>
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                  {new Date(notification.timestamp).toLocaleTimeString()}
                                </p>
                              </div>
                              <button
                                onClick={() => removeNotification(notification.id)}
                                className="flex-shrink-0 text-gray-400 hover:text-gray-500"
                              >
                                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                </svg>
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    </>
                  ) : (
                    <div className="p-6 text-center">
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        No notifications
                      </p>
                    </div>
                  )}
                </div>
                {/* Click outside to close */}
                <div 
                  className="fixed inset-0 z-40" 
                  onClick={() => setShowNotifications(false)}
                />
              </>
            )}
          </div>

          {/* User menu */}
          {user && (
            <div className="relative">
              <button
                onClick={() => setShowUserMenu(!showUserMenu)}
                className="flex items-center space-x-2 p-2 rounded-full text-gray-500 hover:text-gray-900 hover:bg-gray-100 
                         dark:text-gray-400 dark:hover:text-white dark:hover:bg-gray-700 
                         focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <User size={20} />
                <span className="hidden sm:inline text-sm font-medium text-gray-700 dark:text-gray-300">
                  {user.username}
                </span>
              </button>

              {/* User dropdown */}
              {showUserMenu && (
                <>
                  <div className="absolute right-0 mt-2 w-56 bg-white dark:bg-gray-800 
                              rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 
                              z-50">
                    <div className="p-3 border-b border-gray-200 dark:border-gray-700">
                      <div className="flex items-center space-x-2">
                        <User size={16} className="text-gray-500 dark:text-gray-400" />
                        <div>
                          <p className="text-sm font-medium text-gray-900 dark:text-white">
                            {user.username}
                          </p>
                          <p className="text-xs text-gray-500 dark:text-gray-400 capitalize">
                            {user.role}
                          </p>
                        </div>
                      </div>
                    </div>
                    <div className="py-1">
                      <button
                        onClick={handleLogout}
                        className="flex items-center space-x-2 w-full px-3 py-2 text-left text-sm text-gray-700 dark:text-gray-300 
                                 hover:bg-gray-100 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-white"
                      >
                        <LogOut size={16} />
                        <span>Logout</span>
                      </button>
                    </div>
                  </div>
                  {/* Click outside to close */}
                  <div 
                    className="fixed inset-0 z-40" 
                    onClick={() => setShowUserMenu(false)}
                  />
                </>
              )}
            </div>
          )}

          {/* Current time */}
          <div className="hidden sm:block text-sm text-gray-600 dark:text-gray-400">
            {currentTime}
          </div>
        </div>
      </div>
    </header>
  );
} 