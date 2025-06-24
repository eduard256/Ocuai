<script>
  import { onMount } from 'svelte'
  import { systemStats, cameras, events } from '../stores/index.js'
  import { Activity, Camera, AlertTriangle, Clock } from 'lucide-svelte'
  import CameraCard from './CameraCard.svelte'
  import EventCard from './EventCard.svelte'

  let recentEvents = []
  let onlineCameras = []

  $: {
    // Обновляем данные при изменении stores
    recentEvents = $events.slice(0, 5)
    onlineCameras = $cameras.filter(camera => camera.status === 'online').slice(0, 4)
  }

  function formatUptime(seconds) {
    if (!seconds) return '0м'
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    if (hours > 0) {
      return `${hours}ч ${minutes}м`
    }
    return `${minutes}м`
  }

  function getStatusColor(status) {
    switch (status) {
      case 'online': return 'text-success-600 bg-success-100 border-success-200'
      case 'offline': return 'text-gray-600 bg-gray-100 border-gray-200'
      case 'error': return 'text-danger-600 bg-danger-100 border-danger-200'
      default: return 'text-gray-600 bg-gray-100 border-gray-200'
    }
  }
</script>

<div class="space-y-6">
  <!-- Page Header -->
  <div class="flex items-center justify-between">
    <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
      Панель управления
    </h2>
    <div class="text-sm text-gray-500 dark:text-gray-400">
      Последнее обновление: {new Date().toLocaleTimeString('ru-RU')}
    </div>
  </div>

  <!-- Stats Grid -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
    <!-- Total Cameras -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <div class="flex items-center">
        <div class="flex-shrink-0">
          <Camera class="h-8 w-8 text-primary-600" />
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Камеры</p>
          <p class="text-2xl font-semibold text-gray-900 dark:text-white">
            {$systemStats.cameras_total || 0}
          </p>
          <p class="text-xs text-success-600">
            {$systemStats.cameras_online || 0} онлайн
          </p>
        </div>
      </div>
    </div>

    <!-- Today's Events -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <div class="flex items-center">
        <div class="flex-shrink-0">
          <Activity class="h-8 w-8 text-warning-600" />
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">События сегодня</p>
          <p class="text-2xl font-semibold text-gray-900 dark:text-white">
            {$systemStats.events_today || 0}
          </p>
          <p class="text-xs text-gray-600 dark:text-gray-400">
            За последние 24ч
          </p>
        </div>
      </div>
    </div>

    <!-- Total Events -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <div class="flex items-center">
        <div class="flex-shrink-0">
          <AlertTriangle class="h-8 w-8 text-danger-600" />
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Всего событий</p>
          <p class="text-2xl font-semibold text-gray-900 dark:text-white">
            {$systemStats.events_total || 0}
          </p>
          <p class="text-xs text-gray-600 dark:text-gray-400">
            За все время
          </p>
        </div>
      </div>
    </div>

    <!-- System Uptime -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <div class="flex items-center">
        <div class="flex-shrink-0">
          <Clock class="h-8 w-8 text-success-600" />
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Время работы</p>
          <p class="text-2xl font-semibold text-gray-900 dark:text-white">
            {formatUptime($systemStats.system_uptime)}
          </p>
          <p class="text-xs text-success-600">
            Система стабильна
          </p>
        </div>
      </div>
    </div>
  </div>

  <!-- Two Column Layout -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <!-- Active Cameras -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
      <div class="p-6 border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-medium text-gray-900 dark:text-white">
            Активные камеры
          </h3>
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {onlineCameras.length} из {$cameras.length}
          </span>
        </div>
      </div>
      <div class="p-6">
        {#if onlineCameras.length > 0}
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {#each onlineCameras as camera (camera.id)}
              <CameraCard {camera} showControls={false} />
            {/each}
          </div>
          {#if $cameras.length > 4}
            <div class="mt-4 text-center">
              <button 
                class="text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 text-sm font-medium"
                on:click={() => window.dispatchEvent(new CustomEvent('navigate', { detail: 'cameras' }))}
              >
                Показать все камеры ({$cameras.length})
              </button>
            </div>
          {/if}
        {:else}
          <div class="text-center py-8">
            <Camera class="mx-auto h-12 w-12 text-gray-400" />
            <h3 class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
              Нет активных камер
            </h3>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Добавьте камеру для начала мониторинга
            </p>
            <div class="mt-6">
              <button 
                class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
                on:click={() => window.dispatchEvent(new CustomEvent('navigate', { detail: 'cameras' }))}
              >
                <Camera class="-ml-1 mr-2 h-5 w-5" />
                Добавить камеру
              </button>
            </div>
          </div>
        {/if}
      </div>
    </div>

    <!-- Recent Events -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
      <div class="p-6 border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-medium text-gray-900 dark:text-white">
            Последние события
          </h3>
          <span class="text-sm text-gray-500 dark:text-gray-400">
            Топ 5
          </span>
        </div>
      </div>
      <div class="p-6">
        {#if recentEvents.length > 0}
          <div class="space-y-4">
            {#each recentEvents as event (event.id)}
              <EventCard {event} compact={true} />
            {/each}
          </div>
          <div class="mt-4 text-center">
            <button 
              class="text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 text-sm font-medium"
              on:click={() => window.dispatchEvent(new CustomEvent('navigate', { detail: 'events' }))}
            >
              Показать все события
            </button>
          </div>
        {:else}
          <div class="text-center py-8">
            <Activity class="mx-auto h-12 w-12 text-gray-400" />
            <h3 class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
              Нет событий
            </h3>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              События будут отображаться здесь после обнаружения
            </p>
          </div>
        {/if}
      </div>
    </div>
  </div>
</div> 