'use client';

import { useState } from 'react';
import { Save, Bell, Shield, Database, Video } from 'lucide-react';

export default function SettingsPage() {
  const [settings, setSettings] = useState({
    aiEnabled: true,
    aiThreshold: 0.7,
    motionDetection: true,
    telegramEnabled: false,
    retentionDays: 7,
    maxVideoSizeMB: 100,
  });

  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    // Simulate save
    await new Promise(resolve => setTimeout(resolve, 1000));
    setSaving(false);
    alert('Settings saved successfully!');
  };

  return (
    <div className="p-6 max-w-4xl">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Settings</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Configure your Ocuai surveillance system
        </p>
      </div>

      <div className="space-y-6">
        {/* AI Detection Settings */}
        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <div className="flex items-center mb-4">
              <Shield className="h-5 w-5 text-gray-400 mr-2" />
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">AI Detection</h3>
            </div>
            
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <label htmlFor="ai-enabled" className="flex items-center">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Enable AI Detection</span>
                </label>
                <button
                  id="ai-enabled"
                  type="button"
                  onClick={() => setSettings(s => ({ ...s, aiEnabled: !s.aiEnabled }))}
                  className={`${
                    settings.aiEnabled ? 'bg-indigo-600' : 'bg-gray-200'
                  } relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2`}
                >
                  <span
                    className={`${
                      settings.aiEnabled ? 'translate-x-5' : 'translate-x-0'
                    } pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out`}
                  />
                </button>
              </div>

              {settings.aiEnabled && (
                <div>
                  <label htmlFor="ai-threshold" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Detection Threshold
                  </label>
                  <div className="mt-1 flex items-center space-x-3">
                    <input
                      type="range"
                      id="ai-threshold"
                      min="0"
                      max="1"
                      step="0.1"
                      value={settings.aiThreshold}
                      onChange={(e) => setSettings(s => ({ ...s, aiThreshold: parseFloat(e.target.value) }))}
                      className="flex-1"
                    />
                    <span className="text-sm text-gray-500">{(settings.aiThreshold * 100).toFixed(0)}%</span>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Motion Detection */}
        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <Video className="h-5 w-5 text-gray-400 mr-2" />
                <label htmlFor="motion-detection" className="text-lg font-medium text-gray-900 dark:text-white">
                  Motion Detection
                </label>
              </div>
              <button
                id="motion-detection"
                type="button"
                onClick={() => setSettings(s => ({ ...s, motionDetection: !s.motionDetection }))}
                className={`${
                  settings.motionDetection ? 'bg-indigo-600' : 'bg-gray-200'
                } relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2`}
              >
                <span
                  className={`${
                    settings.motionDetection ? 'translate-x-5' : 'translate-x-0'
                  } pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out`}
                />
              </button>
            </div>
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              Detect motion in camera feeds and trigger alerts
            </p>
          </div>
        </div>

        {/* Notifications */}
        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <Bell className="h-5 w-5 text-gray-400 mr-2" />
                <label htmlFor="telegram" className="text-lg font-medium text-gray-900 dark:text-white">
                  Telegram Notifications
                </label>
              </div>
              <button
                id="telegram"
                type="button"
                onClick={() => setSettings(s => ({ ...s, telegramEnabled: !s.telegramEnabled }))}
                className={`${
                  settings.telegramEnabled ? 'bg-indigo-600' : 'bg-gray-200'
                } relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2`}
              >
                <span
                  className={`${
                    settings.telegramEnabled ? 'translate-x-5' : 'translate-x-0'
                  } pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out`}
                />
              </button>
            </div>
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              Send alerts to Telegram when events are detected
            </p>
          </div>
        </div>

        {/* Storage */}
        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <div className="flex items-center mb-4">
              <Database className="h-5 w-5 text-gray-400 mr-2" />
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">Storage</h3>
            </div>
            
            <div className="space-y-4">
              <div>
                <label htmlFor="retention" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Video Retention (days)
                </label>
                <input
                  type="number"
                  id="retention"
                  min="1"
                  max="30"
                  value={settings.retentionDays}
                  onChange={(e) => setSettings(s => ({ ...s, retentionDays: parseInt(e.target.value) }))}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600"
                />
              </div>

              <div>
                <label htmlFor="max-size" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Max Video Size (MB)
                </label>
                <input
                  type="number"
                  id="max-size"
                  min="10"
                  max="1000"
                  value={settings.maxVideoSizeMB}
                  onChange={(e) => setSettings(s => ({ ...s, maxVideoSizeMB: parseInt(e.target.value) }))}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
            </div>
          </div>
        </div>

        {/* Save Button */}
        <div className="flex justify-end">
          <button
            onClick={handleSave}
            disabled={saving}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
          >
            {saving ? (
              <>
                <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Saving...
              </>
            ) : (
              <>
                <Save className="-ml-1 mr-2 h-5 w-5" />
                Save Changes
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
} 