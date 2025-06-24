<script>
  import { Play, Pause, Settings, Trash2, Eye, EyeOff } from 'lucide-svelte'
  
  export let camera
  export let showControls = true
  export let compact = false
  
  let imageError = false
  let isPlaying = false
  
  function handleImageError() {
    imageError = true
  }
  
  function getStatusColor(status) {
    switch (status) {
      case 'online': return 'bg-success-500'
      case 'offline': return 'bg-gray-500'
      case 'error': return 'bg-danger-500'
      default: return 'bg-gray-500'
    }
  }
  
  function getStatusText(status) {
    switch (status) {
      case 'online': return 'Онлайн'
      case 'offline': return 'Офлайн'
      case 'error': return 'Ошибка'
      default: return 'Неизвестно'
    }
  }
  
  function formatLastSeen(lastSeen) {
    if (!lastSeen) return 'Никогда'
    const date = new Date(lastSeen)
    return date.toLocaleString('ru-RU')
  }
  
  async function toggleStream() {
    isPlaying = !isPlaying
  }
  
  async function deleteCamera() {
    if (confirm(`Удалить камеру "${camera.name}"?`)) {
      try {
        const response = await fetch(`/api/cameras/${camera.id}`, {
          method: 'DELETE'
        })
        if (response.ok) {
          // Камера будет удалена через WebSocket уведомление
        } else {
          alert('Ошибка при удалении камеры')
        }
      } catch (error) {
        console.error('Delete camera error:', error)
        alert('Ошибка при удалении камеры')
      }
    }
  }
</script>

<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden
            {compact ? 'min-h-0' : 'min-h-[300px]'}">
  
  <!-- Stream Container -->
  <div class="relative {compact ? 'aspect-video h-32' : 'aspect-video h-48'} bg-gray-900">
    <!-- Live Stream or Snapshot -->
    {#if !imageError}
      <img
        src="/api/streaming/cameras/{camera.id}/snapshot"
        alt="Camera {camera.name}"
        class="w-full h-full object-cover"
        on:error={handleImageError}
      />
    {:else}
      <div class="w-full h-full flex items-center justify-center text-gray-400">
        <div class="text-center">
          <EyeOff size={compact ? 24 : 32} class="mx-auto mb-2" />
          <p class="text-sm">Нет изображения</p>
        </div>
      </div>
    {/if}
    
    <!-- Status Indicator -->
    <div class="absolute top-2 left-2 flex items-center space-x-2">
      <div class="flex items-center space-x-1 bg-black bg-opacity-50 rounded-full px-2 py-1">
        <div class="w-2 h-2 rounded-full {getStatusColor(camera.status)}"></div>
        <span class="text-white text-xs font-medium">
          {getStatusText(camera.status)}
        </span>
      </div>
    </div>
    
    <!-- Detection Indicators -->
    <div class="absolute top-2 right-2 flex space-x-1">
      {#if camera.motion_detection}
        <div class="bg-warning-500 bg-opacity-80 rounded px-2 py-1">
          <span class="text-white text-xs font-medium">Движение</span>
        </div>
      {/if}
      {#if camera.ai_detection}
        <div class="bg-primary-500 bg-opacity-80 rounded px-2 py-1">
          <span class="text-white text-xs font-medium">ИИ</span>
        </div>
      {/if}
    </div>
    
    <!-- Play/Pause Overlay -->
    {#if !compact}
      <div class="absolute inset-0 flex items-center justify-center opacity-0 hover:opacity-100 transition-opacity bg-black bg-opacity-30">
        <button
          on:click={toggleStream}
          class="bg-white bg-opacity-80 hover:bg-opacity-100 rounded-full p-3 transition-all"
        >
          {#if isPlaying}
            <Pause size={24} class="text-gray-800" />
          {:else}
            <Play size={24} class="text-gray-800 ml-1" />
          {/if}
        </button>
      </div>
    {/if}
    
    <!-- Live indicator -->
    {#if camera.status === 'online'}
      <div class="absolute bottom-2 left-2">
        <div class="flex items-center space-x-1 bg-danger-500 rounded px-2 py-1">
          <div class="w-2 h-2 bg-white rounded-full animate-pulse"></div>
          <span class="text-white text-xs font-bold">LIVE</span>
        </div>
      </div>
    {/if}
  </div>
  
  <!-- Camera Info -->
  <div class="p-4">
    <div class="flex items-center justify-between mb-2">
      <h3 class="text-lg font-medium text-gray-900 dark:text-white truncate">
        {camera.name}
      </h3>
      {#if showControls && !compact}
        <div class="flex space-x-1">
          <button
            class="p-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            title="Настройки"
          >
            <Settings size={16} />
          </button>
          <button
            on:click={deleteCamera}
            class="p-1 text-gray-500 hover:text-danger-600 dark:text-gray-400 dark:hover:text-danger-400"
            title="Удалить"
          >
            <Trash2 size={16} />
          </button>
        </div>
      {/if}
    </div>
    
    {#if !compact}
      <!-- Camera Details -->
      <div class="space-y-2 text-sm text-gray-600 dark:text-gray-400">
        <div class="flex justify-between">
          <span>Статус:</span>
          <span class="font-medium {camera.status === 'online' ? 'text-success-600' : 'text-gray-500'}">
            {getStatusText(camera.status)}
          </span>
        </div>
        
        {#if camera.last_seen}
          <div class="flex justify-between">
            <span>Последний раз:</span>
            <span class="font-medium">
              {formatLastSeen(camera.last_seen)}
            </span>
          </div>
        {/if}
        
        <div class="flex justify-between">
          <span>Детекция:</span>
          <div class="flex space-x-2">
            {#if camera.motion_detection}
              <span class="text-warning-600 font-medium">Движение</span>
            {/if}
            {#if camera.ai_detection}
              <span class="text-primary-600 font-medium">ИИ</span>
            {/if}
            {#if !camera.motion_detection && !camera.ai_detection}
              <span class="text-gray-500">Выключена</span>
            {/if}
          </div>
        </div>
      </div>
    {:else}
      <!-- Compact View -->
      <div class="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
        <span class="font-medium {camera.status === 'online' ? 'text-success-600' : 'text-gray-500'}">
          {getStatusText(camera.status)}
        </span>
        <div class="flex space-x-1">
          {#if camera.motion_detection}
            <span class="text-warning-600">M</span>
          {/if}
          {#if camera.ai_detection}
            <span class="text-primary-600">AI</span>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div> 