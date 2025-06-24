<script>
  import { onMount } from 'svelte'
  import { cameras, events, systemStats, websocket } from './stores/index.js'
  import Header from './components/Header.svelte'
  import Sidebar from './components/Sidebar.svelte'
  import Dashboard from './components/Dashboard.svelte'
  import CameraGrid from './components/CameraGrid.svelte'
  import EventsList from './components/EventsList.svelte'
  import Settings from './components/Settings.svelte'
  import { connectWebSocket } from './utils/websocket.js'

  let currentPage = 'dashboard'
  let sidebarOpen = false

  const pages = {
    dashboard: { component: Dashboard, title: 'Панель управления' },
    cameras: { component: CameraGrid, title: 'Камеры' },
    events: { component: EventsList, title: 'События' },
    settings: { component: Settings, title: 'Настройки' }
  }

  function navigateTo(page) {
    currentPage = page
    sidebarOpen = false
  }

  async function loadInitialData() {
    try {
      // Загружаем камеры
      const camerasResponse = await fetch('/api/cameras')
      if (camerasResponse.ok) {
        const camerasData = await camerasResponse.json()
        cameras.set(camerasData.data || [])
      }

      // Загружаем события
      const eventsResponse = await fetch('/api/events?limit=20')
      if (eventsResponse.ok) {
        const eventsData = await eventsResponse.json()
        events.set(eventsData.data || [])
      }

      // Загружаем статистику
      const statsResponse = await fetch('/api/stats')
      if (statsResponse.ok) {
        const statsData = await statsResponse.json()
        systemStats.set(statsData.data || {})
      }
    } catch (error) {
      console.error('Failed to load initial data:', error)
    }
  }

  onMount(() => {
    loadInitialData()
    connectWebSocket()
  })
</script>

<div class="flex h-full bg-gray-50 dark:bg-gray-900">
  <!-- Sidebar -->
  <Sidebar 
    {currentPage} 
    {sidebarOpen}
    on:navigate={(e) => navigateTo(e.detail)}
    on:toggle={() => sidebarOpen = !sidebarOpen}
  />

  <!-- Main content -->
  <div class="flex-1 flex flex-col min-w-0">
    <!-- Header -->
    <Header 
      title={pages[currentPage]?.title || 'Ocuai'}
      on:toggleSidebar={() => sidebarOpen = !sidebarOpen}
    />

    <!-- Page content -->
    <main class="flex-1 overflow-auto p-6">
      {#if pages[currentPage]}
        <svelte:component this={pages[currentPage].component} />
      {:else}
        <div class="text-center py-12">
          <p class="text-gray-500 dark:text-gray-400">Страница не найдена</p>
        </div>
      {/if}
    </main>
  </div>
</div>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  }

  :global(.dark) {
    color-scheme: dark;
  }
</style> 