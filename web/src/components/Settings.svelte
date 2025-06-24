<script>
  import { onMount } from 'svelte'
  import { addNotification } from '../stores/index.js'
  import { Settings as SettingsIcon, Save, Trash2, Download, Upload } from 'lucide-svelte'

  let settings = {
    telegram: {
      token: '',
      allowed_users: '',
      notification_hours: '08:00-22:00'
    },
    ai: {
      enabled: true,
      threshold: 0.7,
      device_type: 'cpu'
    },
    storage: {
      retention_days: 7,
      max_video_size_mb: 50
    },
    system: {
      auto_cleanup: true,
      debug_logging: false
    }
  }

  let loading = false
  let saving = false

  onMount(async () => {
    await loadSettings()
  })

  async function loadSettings() {
    loading = true
    try {
      const response = await fetch('/api/settings')
      if (response.ok) {
        const result = await response.json()
        if (result.success && result.data) {
          settings = { ...settings, ...result.data }
        }
      }
    } catch (error) {
      console.error('Load settings error:', error)
      addNotification('Ошибка загрузки настроек', 'error')
    } finally {
      loading = false
    }
  }

  async function saveSettings() {
    saving = true
    try {
      const response = await fetch('/api/settings', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(settings)
      })

      if (response.ok) {
        addNotification('Настройки сохранены', 'success')
      } else {
        const error = await response.json()
        addNotification(`Ошибка: ${error.error || 'Не удалось сохранить настройки'}`, 'error')
      }
    } catch (error) {
      console.error('Save settings error:', error)
      addNotification('Ошибка сохранения настроек', 'error')
    } finally {
      saving = false
    }
  }

  async function clearAllData() {
    if (confirm('Это действие удалит ВСЕ данные системы (камеры, события, видео). Продолжить?')) {
      try {
        const response = await fetch('/api/system/clear', {
          method: 'POST'
        })
        if (response.ok) {
          addNotification('Данные очищены', 'success')
          // Перезагрузка страницы через 2 секунды
          setTimeout(() => window.location.reload(), 2000)
        } else {
          addNotification('Ошибка очистки данных', 'error')
        }
      } catch (error) {
        console.error('Clear data error:', error)
        addNotification('Ошибка очистки данных', 'error')
      }
    }
  }

  async function exportSettings() {
    try {
      const response = await fetch('/api/settings/export')
      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `ocuai-settings-${new Date().toISOString().split('T')[0]}.json`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
        addNotification('Настройки экспортированы', 'success')
      }
    } catch (error) {
      console.error('Export settings error:', error)
      addNotification('Ошибка экспорта настроек', 'error')
    }
  }

  function handleFileImport(event) {
    const file = event.target.files[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = async (e) => {
      try {
        const importedSettings = JSON.parse(e.target.result)
        settings = { ...settings, ...importedSettings }
        addNotification('Настройки импортированы', 'success')
      } catch (error) {
        addNotification('Ошибка импорта настроек', 'error')
      }
    }
    reader.readAsText(file)
  }
</script>

<div class="space-y-6">
  <!-- Page Header -->
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
        Настройки
      </h2>
      <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
        Конфигурация системы Ocuai
      </p>
    </div>
    
    <div class="flex space-x-3">
      <input
        type="file"
        accept=".json"
        on:change={handleFileImport}
        class="hidden"
        id="import-file"
      />
      <label
        for="import-file"
        class="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 shadow-sm text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 cursor-pointer"
      >
        <Upload size={16} class="mr-2" />
        Импорт
      </label>
      <button
        on:click={exportSettings}
        class="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 shadow-sm text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
      >
        <Download size={16} class="mr-2" />
        Экспорт
      </button>
      <button
        on:click={saveSettings}
        disabled={saving}
        class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50"
      >
        <Save size={16} class="mr-2" />
        {saving ? 'Сохранение...' : 'Сохранить'}
      </button>
    </div>
  </div>

  {#if loading}
    <div class="text-center py-12">
      <div class="inline-flex items-center text-sm text-gray-600 dark:text-gray-400">
        <svg class="animate-spin -ml-1 mr-3 h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        Загрузка настроек...
      </div>
    </div>
  {:else}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Telegram Settings -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">
          Telegram Bot
        </h3>
        
        <div class="space-y-4">
                     <div>
             <label for="telegram-token" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
               Bot Token
             </label>
             <input
               id="telegram-token"
               type="password"
               bind:value={settings.telegram.token}
              placeholder="1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
              class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
          
                     <div>
             <label for="telegram-users" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
               Разрешенные пользователи (ID через запятую)
             </label>
             <input
               id="telegram-users"
               type="text"
               bind:value={settings.telegram.allowed_users}
              placeholder="123456789,987654321"
              class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
          
                     <div>
             <label for="notification-hours" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
               Часы уведомлений
             </label>
             <input
               id="notification-hours"
               type="text"
               bind:value={settings.telegram.notification_hours}
              placeholder="08:00-22:00"
              class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
        </div>
      </div>

      <!-- AI Settings -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">
          Искусственный интеллект
        </h3>
        
        <div class="space-y-4">
          <label class="flex items-center space-x-3">
            <input
              type="checkbox"
              bind:checked={settings.ai.enabled}
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">Включить ИИ детекцию</span>
          </label>
          
                     <div>
             <label for="ai-threshold" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
               Порог уверенности: {Math.round(settings.ai.threshold * 100)}%
             </label>
             <input
               id="ai-threshold"
               type="range"
               min="0.1"
               max="1.0"
               step="0.1"
               bind:value={settings.ai.threshold}
              class="w-full h-2 bg-gray-200 dark:bg-gray-600 rounded-lg appearance-none cursor-pointer"
            />
          </div>
          
                     <div>
             <label for="ai-device" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
               Устройство вычислений
             </label>
             <select
               id="ai-device"
               bind:value={settings.ai.device_type}
              class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="cpu">CPU</option>
              <option value="gpu">GPU (CUDA)</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Storage Settings -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">
          Хранилище
        </h3>
        
                 <div class="space-y-4">
           <div>
             <label for="retention-days" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
               Хранить записи (дней)
             </label>
             <input
               id="retention-days"
               type="number"
               min="1"
               max="365"
               bind:value={settings.storage.retention_days}
              class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
          
                     <div>
             <label for="video-size" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
               Максимальный размер видео (МБ)
             </label>
             <input
               id="video-size"
               type="number"
               min="10"
               max="1000"
               bind:value={settings.storage.max_video_size_mb}
              class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
        </div>
      </div>

      <!-- System Settings -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">
          Система
        </h3>
        
        <div class="space-y-4">
          <label class="flex items-center space-x-3">
            <input
              type="checkbox"
              bind:checked={settings.system.auto_cleanup}
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">Автоматическая очистка старых файлов</span>
          </label>
          
          <label class="flex items-center space-x-3">
            <input
              type="checkbox"
              bind:checked={settings.system.debug_logging}
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">Включить отладочные логи</span>
          </label>
        </div>
      </div>
    </div>

    <!-- Danger Zone -->
    <div class="bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800 p-6">
      <h3 class="text-lg font-medium text-red-900 dark:text-red-300 mb-4">
        Опасная зона
      </h3>
      <p class="text-sm text-red-700 dark:text-red-400 mb-4">
        Эти действия необратимы. Будьте осторожны!
      </p>
      <button
        on:click={clearAllData}
        class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
      >
        <Trash2 size={16} class="mr-2" />
        Очистить все данные
      </button>
    </div>
  {/if}
</div> 