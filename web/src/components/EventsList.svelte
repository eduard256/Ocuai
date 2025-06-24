<script>
  import { onMount } from 'svelte'
  import { events, cameras } from '../stores/index.js'
  import { Calendar, Search, Download, Filter } from 'lucide-svelte'
  import EventCard from './EventCard.svelte'

  let searchQuery = ''
  let cameraFilter = 'all'
  let typeFilter = 'all'
  let dateFilter = 'all'
  let filteredEvents = []
  let loading = false
  let page = 1
  let hasMore = true

  $: {
    filteredEvents = $events.filter(event => {
      const matchesSearch = event.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                           event.camera_name.toLowerCase().includes(searchQuery.toLowerCase())
      
      const matchesCamera = cameraFilter === 'all' || event.camera_id === cameraFilter
      const matchesType = typeFilter === 'all' || event.type === typeFilter
      
      let matchesDate = true
      if (dateFilter !== 'all') {
        const eventDate = new Date(event.created_at)
        const now = new Date()
        
        switch (dateFilter) {
          case 'today':
            matchesDate = eventDate.toDateString() === now.toDateString()
            break
          case 'week':
            const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
            matchesDate = eventDate >= weekAgo
            break
          case 'month':
            const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
            matchesDate = eventDate >= monthAgo
            break
        }
      }
      
      return matchesSearch && matchesCamera && matchesType && matchesDate
    })
  }

  async function loadMoreEvents() {
    if (loading || !hasMore) return
    
    loading = true
    try {
      const response = await fetch(`/api/events?limit=20&offset=${page * 20}`)
      if (response.ok) {
        const result = await response.json()
        const newEvents = result.data || []
        
        if (newEvents.length === 0) {
          hasMore = false
        } else {
          events.update(items => [...items, ...newEvents])
          page++
        }
      }
    } catch (error) {
      console.error('Load events error:', error)
    } finally {
      loading = false
    }
  }

  async function exportEvents() {
    try {
      const params = new URLSearchParams()
      if (cameraFilter !== 'all') params.append('camera_id', cameraFilter)
      if (typeFilter !== 'all') params.append('type', typeFilter)
      
      const response = await fetch(`/api/events/export?${params}`)
      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `events-${new Date().toISOString().split('T')[0]}.csv`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
      }
    } catch (error) {
      console.error('Export events error:', error)
    }
  }

  function clearFilters() {
    searchQuery = ''
    cameraFilter = 'all'  
    typeFilter = 'all'
    dateFilter = 'all'
  }
</script>

<div class="space-y-6">
  <!-- Page Header -->
  <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between">
    <div>
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
        События
      </h2>
      <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
        История событий системы безопасности
      </p>
    </div>
    
    <div class="mt-4 sm:mt-0 flex space-x-3">
      <button
        on:click={exportEvents}
        class="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 shadow-sm text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
      >
        <Download size={16} class="mr-2" />
        Экспорт
      </button>
    </div>
  </div>

  <!-- Filters -->
  <div class="bg-white dark:bg-gray-800 p-4 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <!-- Search -->
      <div class="relative">
        <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
          <Search size={16} class="text-gray-400" />
        </div>
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Поиск событий..."
          class="block w-full pl-10 pr-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md leading-5 bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 sm:text-sm"
        />
      </div>

      <!-- Camera Filter -->
      <div>
        <select
          bind:value={cameraFilter}
          class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 sm:text-sm"
        >
          <option value="all">Все камеры</option>
          {#each $cameras as camera (camera.id)}
            <option value={camera.id}>{camera.name}</option>
          {/each}
        </select>
      </div>

      <!-- Type Filter -->
      <div>
        <select
          bind:value={typeFilter}
          class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 sm:text-sm"
        >
          <option value="all">Все типы</option>
          <option value="motion">Движение</option>
          <option value="ai_detection">ИИ детекция</option>
        </select>
      </div>

      <!-- Date Filter -->
      <div>
        <select
          bind:value={dateFilter}
          class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 sm:text-sm"
        >
          <option value="all">Все время</option>
          <option value="today">Сегодня</option>
          <option value="week">Неделя</option>
          <option value="month">Месяц</option>
        </select>
      </div>
    </div>
    
    <!-- Filter Actions -->
    <div class="mt-4 flex items-center justify-between">
      <div class="text-sm text-gray-600 dark:text-gray-400">
        Показано: {filteredEvents.length} из {$events.length} событий
      </div>
      
      {#if searchQuery || cameraFilter !== 'all' || typeFilter !== 'all' || dateFilter !== 'all'}
        <button
          on:click={clearFilters}
          class="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 font-medium"
        >
          Сбросить фильтры
        </button>
      {/if}
    </div>
  </div>

  <!-- Events List -->
  {#if filteredEvents.length > 0}
    <div class="space-y-4">
      {#each filteredEvents as event (event.id)}
        <EventCard {event} compact={false} />
      {/each}
    </div>
    
    <!-- Load More -->
    {#if hasMore && !loading}
      <div class="text-center">
        <button
          on:click={loadMoreEvents}
          class="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 shadow-sm text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
        >
          Загрузить еще
        </button>
      </div>
    {:else if loading}
      <div class="text-center py-4">
        <div class="inline-flex items-center text-sm text-gray-600 dark:text-gray-400">
          <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-primary-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          Загрузка...
        </div>
      </div>
    {/if}
  {:else if $events.length > 0}
    <!-- No events match filters -->
    <div class="text-center py-12">
      <Filter class="mx-auto h-12 w-12 text-gray-400" />
      <h3 class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
        События не найдены
      </h3>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Попробуйте изменить параметры фильтрации
      </p>
      <div class="mt-6">
        <button
          on:click={clearFilters}
          class="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 shadow-sm text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
        >
          Сбросить фильтры
        </button>
      </div>
    </div>
  {:else}
    <!-- No events at all -->
    <div class="text-center py-12">
      <Calendar class="mx-auto h-12 w-12 text-gray-400" />
      <h3 class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
        Нет событий
      </h3>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        События будут отображаться здесь после их обнаружения
      </p>
    </div>
  {/if}
</div> 