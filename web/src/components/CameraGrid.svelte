<script>
  import { onMount } from 'svelte'
  import { cameras, addNotification } from '../stores/index.js'
  import { Plus, Search, Filter } from 'lucide-svelte'
  import CameraCard from './CameraCard.svelte'
  import AddCameraModal from './AddCameraModal.svelte'

  let searchQuery = ''
  let statusFilter = 'all'
  let showAddModal = false
  let filteredCameras = []

  $: {
    filteredCameras = $cameras.filter(camera => {
      const matchesSearch = camera.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                           camera.id.toLowerCase().includes(searchQuery.toLowerCase())
      
      const matchesStatus = statusFilter === 'all' || camera.status === statusFilter
      
      return matchesSearch && matchesStatus
    })
  }

  function openAddModal() {
    showAddModal = true
  }

  function closeAddModal() {
    showAddModal = false
  }

  async function onCameraAdded(cameraData) {
    try {
      const response = await fetch('/api/cameras', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(cameraData)
      })

      if (response.ok) {
        const result = await response.json()
        cameras.update(items => [...items, result.data])
        addNotification(`Камера "${cameraData.name}" добавлена`, 'success')
        closeAddModal()
      } else {
        const error = await response.json()
        addNotification(`Ошибка: ${error.error || 'Не удалось добавить камеру'}`, 'error')
      }
    } catch (error) {
      console.error('Add camera error:', error)
      addNotification('Ошибка при добавлении камеры', 'error')
    }
  }
</script>

<div class="space-y-6">
  <!-- Page Header -->
  <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between">
    <div>
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
        Камеры
      </h2>
      <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
        Управление камерами видеонаблюдения
      </p>
    </div>
    
    <div class="mt-4 sm:mt-0">
      <button
        on:click={openAddModal}
        class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
      >
        <Plus size={16} class="mr-2" />
        Добавить камеру
      </button>
    </div>
  </div>

  <!-- Filters and Search -->
  <div class="bg-white dark:bg-gray-800 p-4 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between space-y-3 sm:space-y-0 sm:space-x-4">
      <!-- Search -->
      <div class="relative flex-1 max-w-md">
        <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
          <Search size={16} class="text-gray-400" />
        </div>
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Поиск камер..."
          class="block w-full pl-10 pr-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md leading-5 bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 sm:text-sm"
        />
      </div>

      <!-- Status Filter -->
      <div class="flex items-center space-x-2">
        <Filter size={16} class="text-gray-400" />
        <select
          bind:value={statusFilter}
          class="block px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 sm:text-sm"
        >
          <option value="all">Все статусы</option>
          <option value="online">Онлайн</option>
          <option value="offline">Офлайн</option>
          <option value="error">Ошибка</option>
        </select>
      </div>

      <!-- Stats -->
      <div class="text-sm text-gray-600 dark:text-gray-400">
        Показано: {filteredCameras.length} из {$cameras.length}
      </div>
    </div>
  </div>

  <!-- Camera Grid -->
  {#if filteredCameras.length > 0}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
      {#each filteredCameras as camera (camera.id)}
        <CameraCard {camera} showControls={true} />
      {/each}
    </div>
  {:else if $cameras.length > 0}
    <!-- No cameras match filters -->
    <div class="text-center py-12">
      <Search class="mx-auto h-12 w-12 text-gray-400" />
      <h3 class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
        Камеры не найдены
      </h3>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Попробуйте изменить параметры поиска
      </p>
      <div class="mt-6">
        <button
          on:click={() => { searchQuery = ''; statusFilter = 'all' }}
          class="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 shadow-sm text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
        >
          Сбросить фильтры
        </button>
      </div>
    </div>
  {:else}
    <!-- No cameras at all -->
    <div class="text-center py-12">
      <div class="mx-auto h-12 w-12 bg-gray-400 rounded-lg flex items-center justify-center">
        <Plus size={24} class="text-white" />
      </div>
      <h3 class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
        Нет камер
      </h3>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Добавьте первую камеру для начала мониторинга
      </p>
      <div class="mt-6">
        <button
          on:click={openAddModal}
          class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
        >
          <Plus size={16} class="mr-2" />
          Добавить камеру
        </button>
      </div>
    </div>
  {/if}
</div>

<!-- Add Camera Modal -->
{#if showAddModal}
  <AddCameraModal 
    on:close={closeAddModal}
    on:save={e => onCameraAdded(e.detail)}
  />
{/if} 