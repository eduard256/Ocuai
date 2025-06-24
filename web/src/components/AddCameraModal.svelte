<script>
  import { createEventDispatcher } from 'svelte'
  import { X, Camera, Loader } from 'lucide-svelte'

  const dispatch = createEventDispatcher()

  let formData = {
    name: '',
    rtsp_url: '',
    username: '',
    password: '',
    motion_detection: true,
    ai_detection: true,
    sensitivity: 0.7,
    record_motion: true,
    send_telegram: true
  }

  let isTestingConnection = false
  let testResult = null
  let errors = {}

  function closeModal() {
    dispatch('close')
  }

  function validateForm() {
    errors = {}
    
    if (!formData.name.trim()) {
      errors.name = 'Название обязательно'
    }
    
    if (!formData.rtsp_url.trim()) {
      errors.rtsp_url = 'RTSP URL обязателен'
    } else if (!isValidRTSPUrl(formData.rtsp_url)) {
      errors.rtsp_url = 'Некорректный RTSP URL'
    }
    
    return Object.keys(errors).length === 0
  }

  function isValidRTSPUrl(url) {
    try {
      const parsed = new URL(url)
      return parsed.protocol === 'rtsp:'
    } catch {
      return false
    }
  }

  async function testConnection() {
    if (!formData.rtsp_url.trim()) {
      testResult = { success: false, message: 'Введите RTSP URL' }
      return
    }

    isTestingConnection = true
    testResult = null

    try {
      const response = await fetch('/api/cameras/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          rtsp_url: formData.rtsp_url,
          username: formData.username,
          password: formData.password
        })
      })

      const result = await response.json()
      testResult = result
    } catch (error) {
      testResult = { success: false, message: 'Ошибка соединения с сервером' }
    } finally {
      isTestingConnection = false
    }
  }

  async function saveCamera() {
    if (!validateForm()) return
    
    dispatch('save', formData)
  }

  function handleKeydown(event) {
    if (event.key === 'Escape') {
      closeModal()
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- Modal backdrop -->
<div class="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4">
  <!-- Modal content -->
  <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full max-h-full overflow-y-auto">
    <!-- Modal header -->
    <div class="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
      <div class="flex items-center space-x-3">
        <Camera size={24} class="text-primary-600" />
        <h3 class="text-lg font-medium text-gray-900 dark:text-white">
          Добавить камеру
        </h3>
      </div>
      <button
        on:click={closeModal}
        class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
      >
        <X size={20} />
      </button>
    </div>

    <!-- Modal body -->
    <div class="p-6 space-y-6">
      <!-- Camera Name -->
      <div>
        <label for="name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          Название камеры
        </label>
        <input
          id="name"
          type="text"
          bind:value={formData.name}
          placeholder="Например: Входная дверь"
          class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          class:border-red-500={errors.name}
        />
        {#if errors.name}
          <p class="mt-1 text-sm text-red-600">{errors.name}</p>
        {/if}
      </div>

      <!-- RTSP URL -->
      <div>
        <label for="rtsp_url" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          RTSP URL
        </label>
        <input
          id="rtsp_url"
          type="text"
          bind:value={formData.rtsp_url}
          placeholder="rtsp://192.168.1.100:554/stream"
          class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          class:border-red-500={errors.rtsp_url}
        />
        {#if errors.rtsp_url}
          <p class="mt-1 text-sm text-red-600">{errors.rtsp_url}</p>
        {/if}
      </div>

      <!-- Authentication (Optional) -->
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="username" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Логин (опционально)
          </label>
          <input
            id="username"
            type="text"
            bind:value={formData.username}
            placeholder="admin"
            class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          />
        </div>
        <div>
          <label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Пароль (опционально)
          </label>
          <input
            id="password"
            type="password"
            bind:value={formData.password}
            placeholder="••••••••"
            class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          />
        </div>
      </div>

      <!-- Connection Test -->
      <div>
        <button
          on:click={testConnection}
          disabled={isTestingConnection || !formData.rtsp_url.trim()}
          class="w-full flex items-center justify-center px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {#if isTestingConnection}
            <Loader size={16} class="mr-2 animate-spin" />
            Тестирование...
          {:else}
            Проверить соединение
          {/if}
        </button>
        
        {#if testResult}
          <div class="mt-2 p-3 rounded-md {testResult.success ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}">
            <p class="text-sm">
              {testResult.message || (testResult.success ? 'Соединение успешно!' : 'Ошибка соединения')}
            </p>
          </div>
        {/if}
      </div>

      <!-- Detection Settings -->
      <div class="space-y-4">
        <h4 class="text-sm font-medium text-gray-900 dark:text-white">Настройки детекции</h4>
        
        <div class="space-y-3">
          <!-- Motion Detection -->
          <label class="flex items-center space-x-3">
            <input
              type="checkbox"
              bind:checked={formData.motion_detection}
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">Детекция движения</span>
          </label>

          <!-- AI Detection -->
          <label class="flex items-center space-x-3">
            <input
              type="checkbox"
              bind:checked={formData.ai_detection}
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">ИИ детекция объектов</span>
          </label>

          <!-- Record Motion -->
          <label class="flex items-center space-x-3">
            <input
              type="checkbox"
              bind:checked={formData.record_motion}
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">Записывать видео при событиях</span>
          </label>

          <!-- Telegram Notifications -->
          <label class="flex items-center space-x-3">
            <input
              type="checkbox"
              bind:checked={formData.send_telegram}
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">Уведомления в Telegram</span>
          </label>
        </div>

        <!-- Sensitivity Slider -->
        <div>
          <label for="sensitivity" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Чувствительность: {Math.round(formData.sensitivity * 100)}%
          </label>
          <input
            id="sensitivity"
            type="range"
            min="0.1"
            max="1.0"
            step="0.1"
            bind:value={formData.sensitivity}
            class="w-full h-2 bg-gray-200 dark:bg-gray-600 rounded-lg appearance-none cursor-pointer"
          />
          <div class="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-1">
            <span>Низкая</span>
            <span>Высокая</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal footer -->
    <div class="flex items-center justify-end space-x-3 p-6 border-t border-gray-200 dark:border-gray-700">
      <button
        on:click={closeModal}
        class="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
      >
        Отмена
      </button>
      <button
        on:click={saveCamera}
        class="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
      >
        Добавить камеру
      </button>
    </div>
  </div>
</div> 