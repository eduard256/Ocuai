<script>
  import { Activity, Eye, Download, Trash2 } from 'lucide-svelte'
  
  export let event
  export let compact = false
  
  function getEventIcon(type) {
    return Activity
  }
  
  function getEventColor(type) {
    switch (type) {
      case 'motion': return 'text-warning-600 bg-warning-100 border-warning-200'
      case 'ai_detection': return 'text-primary-600 bg-primary-100 border-primary-200'
      default: return 'text-gray-600 bg-gray-100 border-gray-200'
    }
  }
  
  function getEventText(type) {
    switch (type) {
      case 'motion': return 'Движение'
      case 'ai_detection': return 'ИИ детекция'
      default: return 'Событие'
    }
  }
  
  function formatDate(dateString) {
    const date = new Date(dateString)
    return date.toLocaleString('ru-RU')
  }
  
  function formatTime(dateString) {
    const date = new Date(dateString)
    return date.toLocaleTimeString('ru-RU')
  }
  
  async function downloadVideo() {
    if (event.video_path) {
      window.open(event.video_path, '_blank')
    }
  }
  
  async function deleteEvent() {
    if (confirm('Удалить это событие?')) {
      try {
        const response = await fetch(`/api/events/${event.id}`, {
          method: 'DELETE'
        })
        if (response.ok) {
          // Событие будет удалено через обновление списка
        } else {
          alert('Ошибка при удалении события')
        }
      } catch (error) {
        console.error('Delete event error:', error)
        alert('Ошибка при удалении события')
      }
    }
  }
</script>

<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden
            {compact ? '' : 'mb-4'}">
  
  <div class="p-4">
    <!-- Event Header -->
    <div class="flex items-start justify-between mb-3">
      <div class="flex items-center space-x-3">
        <!-- Event Type Badge -->
        <div class="flex items-center space-x-1 px-2 py-1 rounded-full text-xs font-medium border
                    {getEventColor(event.type)}">
          <svelte:component this={getEventIcon(event.type)} size={12} />
          <span>{getEventText(event.type)}</span>
        </div>
        
        {#if event.confidence}
          <div class="text-sm text-gray-600 dark:text-gray-400">
            {Math.round(event.confidence * 100)}%
          </div>
        {/if}
      </div>
      
      <!-- Actions -->
      {#if !compact}
        <div class="flex space-x-1">
          {#if event.video_path}
            <button
              on:click={downloadVideo}
              class="p-1 text-gray-500 hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-400"
              title="Скачать видео"
            >
              <Download size={16} />
            </button>
          {/if}
          <button
            class="p-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            title="Подробности"
          >
            <Eye size={16} />
          </button>
          <button
            on:click={deleteEvent}
            class="p-1 text-gray-500 hover:text-danger-600 dark:text-gray-400 dark:hover:text-danger-400"
            title="Удалить"
          >
            <Trash2 size={16} />
          </button>
        </div>
      {/if}
    </div>
    
    <!-- Event Content -->
    <div class="flex space-x-4">
      <!-- Thumbnail -->
      {#if event.thumbnail_path && !compact}
        <div class="flex-shrink-0">
          <img
            src={event.thumbnail_path}
            alt="Event thumbnail"
            class="w-20 h-16 object-cover rounded border border-gray-200 dark:border-gray-600"
          />
        </div>
      {/if}
      
      <!-- Event Details -->
      <div class="flex-1 min-w-0">
        <!-- Camera and Description -->
        <div class="mb-2">
          <h4 class="text-sm font-medium text-gray-900 dark:text-white">
            {event.camera_name}
          </h4>
          {#if event.description}
            <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
              {event.description}
            </p>
          {/if}
        </div>
        
        <!-- Timestamp -->
        <div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
          <span>
            {compact ? formatTime(event.created_at) : formatDate(event.created_at)}
          </span>
          
          {#if !event.processed}
            <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
              Обработка
            </span>
          {/if}
        </div>
      </div>
    </div>
  </div>
  
  <!-- Video Preview (for non-compact) -->
  {#if !compact && event.video_path}
    <div class="border-t border-gray-200 dark:border-gray-700 p-4">
      <div class="flex items-center justify-between">
        <span class="text-sm text-gray-600 dark:text-gray-400">
          Видеозапись доступна
        </span>
        <button
          on:click={downloadVideo}
          class="inline-flex items-center px-3 py-1 border border-transparent text-sm font-medium rounded-md text-primary-700 bg-primary-100 hover:bg-primary-200 dark:bg-primary-900 dark:text-primary-300 dark:hover:bg-primary-800"
        >
          <Download size={14} class="mr-1" />
          Скачать
        </button>
      </div>
    </div>
  {/if}
</div> 