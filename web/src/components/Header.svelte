<script>
  import { createEventDispatcher } from 'svelte'
  import { systemStats, notifications } from '../stores/index.js'
  import { Menu, Bell, Wifi, WifiOff } from 'lucide-svelte'

  export let title = 'Ocuai'

  const dispatch = createEventDispatcher()

  let showNotifications = false
  let isOnline = true

  function toggleSidebar() {
    dispatch('toggleSidebar')
  }

  function toggleNotifications() {
    showNotifications = !showNotifications
  }

  // Проверка соединения
  function checkConnection() {
    isOnline = navigator.onLine
  }

  // Следим за изменениями в сети
  if (typeof window !== 'undefined') {
    window.addEventListener('online', checkConnection)
    window.addEventListener('offline', checkConnection)
  }

  function formatUptime(seconds) {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    return `${hours}ч ${minutes}м`
  }
</script>

<header class="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700">
  <div class="flex items-center justify-between px-6 py-4">
    <!-- Left side -->
    <div class="flex items-center space-x-4">
      <!-- Mobile menu button -->
      <button
        on:click={toggleSidebar}
        class="p-2 rounded-md text-gray-500 hover:text-gray-900 hover:bg-gray-100 
               dark:text-gray-400 dark:hover:text-white dark:hover:bg-gray-700 
               focus:outline-none focus:ring-2 focus:ring-primary-500 lg:hidden"
      >
        <Menu size={20} />
      </button>

      <!-- Title -->
      <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
        {title}
      </h1>

      <!-- Status indicators -->
      <div class="hidden sm:flex items-center space-x-3">
        <!-- Connection status -->
        <div class="flex items-center space-x-1">
          {#if isOnline}
            <Wifi size={16} class="text-success-500" />
            <span class="text-sm text-success-600 dark:text-success-400">Онлайн</span>
          {:else}
            <WifiOff size={16} class="text-danger-500" />
            <span class="text-sm text-danger-600 dark:text-danger-400">Офлайн</span>
          {/if}
        </div>

        <!-- System stats -->
        {#if $systemStats.cameras_total > 0}
          <div class="flex items-center space-x-4 text-sm text-gray-600 dark:text-gray-400">
            <span>
              Камеры: {$systemStats.cameras_online}/{$systemStats.cameras_total}
            </span>
            <span class="hidden md:inline">
              События: {$systemStats.events_today}
            </span>
            {#if $systemStats.system_uptime}
              <span class="hidden lg:inline">
                Uptime: {formatUptime($systemStats.system_uptime)}
              </span>
            {/if}
          </div>
        {/if}
      </div>
    </div>

    <!-- Right side -->
    <div class="flex items-center space-x-3">
      <!-- Notifications -->
      <div class="relative">
        <button
          on:click={toggleNotifications}
          class="p-2 rounded-full text-gray-500 hover:text-gray-900 hover:bg-gray-100 
                 dark:text-gray-400 dark:hover:text-white dark:hover:bg-gray-700 
                 focus:outline-none focus:ring-2 focus:ring-primary-500 relative"
        >
          <Bell size={20} />
          {#if $notifications.length > 0}
            <span class="absolute -top-1 -right-1 inline-flex items-center justify-center 
                         px-2 py-1 text-xs font-bold leading-none text-white 
                         bg-danger-500 rounded-full">
              {$notifications.length}
            </span>
          {/if}
        </button>

        <!-- Notifications dropdown -->
        {#if showNotifications}
          <div class="absolute right-0 mt-2 w-80 bg-white dark:bg-gray-800 
                      rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 
                      z-50 max-h-96 overflow-auto">
            {#if $notifications.length > 0}
              <div class="p-3 border-b border-gray-200 dark:border-gray-700">
                <h3 class="text-sm font-medium text-gray-900 dark:text-white">
                  Уведомления
                </h3>
              </div>
              <div class="divide-y divide-gray-200 dark:divide-gray-700">
                {#each $notifications as notification (notification.id)}
                  <div class="p-3 hover:bg-gray-50 dark:hover:bg-gray-700">
                    <div class="flex items-start space-x-3">
                      <div class="flex-shrink-0">
                        <div class="w-2 h-2 rounded-full bg-{notification.type === 'error' ? 'danger' : notification.type === 'warning' ? 'warning' : 'primary'}-500 mt-2"></div>
                      </div>
                      <div class="flex-1 min-w-0">
                        <p class="text-sm text-gray-900 dark:text-white">
                          {notification.message}
                        </p>
                        <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                          {new Date(notification.timestamp).toLocaleTimeString()}
                        </p>
                      </div>
                    </div>
                  </div>
                {/each}
              </div>
            {:else}
              <div class="p-6 text-center">
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  Нет уведомлений
                </p>
              </div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Current time -->
      <div class="hidden sm:block text-sm text-gray-600 dark:text-gray-400">
        {new Date().toLocaleTimeString('ru-RU')}
      </div>
    </div>
  </div>
</header>

<!-- Click outside to close notifications -->
{#if showNotifications}
  <div 
    class="fixed inset-0 z-40" 
    on:click={() => showNotifications = false}
    role="button"
    tabindex="0"
    on:keydown={() => {}}
  ></div>
{/if} 