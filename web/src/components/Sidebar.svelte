<script>
  import { createEventDispatcher } from 'svelte'
  import { Home, Camera, Activity, Settings, X } from 'lucide-svelte'

  export let currentPage = 'dashboard'
  export let sidebarOpen = false

  const dispatch = createEventDispatcher()

  const menuItems = [
    { id: 'dashboard', label: 'Панель', icon: Home },
    { id: 'cameras', label: 'Камеры', icon: Camera },
    { id: 'events', label: 'События', icon: Activity },
    { id: 'settings', label: 'Настройки', icon: Settings }
  ]

  function navigate(page) {
    dispatch('navigate', page)
  }

  function closeSidebar() {
    dispatch('toggle')
  }
</script>

<!-- Mobile backdrop -->
{#if sidebarOpen}
  <div 
    class="fixed inset-0 bg-black bg-opacity-50 z-40 lg:hidden"
    on:click={closeSidebar}
    role="button"
    tabindex="0"
    on:keydown={() => {}}
  ></div>
{/if}

<!-- Sidebar -->
<div class="fixed inset-y-0 left-0 z-50 w-64 bg-white dark:bg-gray-800 shadow-lg
            transform transition-transform duration-300 ease-in-out lg:translate-x-0 lg:static lg:inset-0
            {sidebarOpen ? 'translate-x-0' : '-translate-x-full'}">
  
  <!-- Sidebar header -->
  <div class="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
    <div class="flex items-center space-x-3">
      <div class="w-8 h-8 bg-primary-500 rounded-lg flex items-center justify-center">
        <Camera size={20} class="text-white" />
      </div>
      <span class="text-xl font-bold text-gray-900 dark:text-white">Ocuai</span>
    </div>
    
    <!-- Close button (mobile only) -->
    <button
      on:click={closeSidebar}
      class="p-2 rounded-md text-gray-500 hover:text-gray-900 hover:bg-gray-100 
             dark:text-gray-400 dark:hover:text-white dark:hover:bg-gray-700 
             lg:hidden"
    >
      <X size={20} />
    </button>
  </div>

  <!-- Navigation -->
  <nav class="mt-6 px-3">
    <ul class="space-y-2">
      {#each menuItems as item (item.id)}
        <li>
          <button
            on:click={() => navigate(item.id)}
            class="w-full flex items-center px-3 py-2 text-sm font-medium rounded-lg
                   transition-colors duration-200 group
                   {currentPage === item.id 
                     ? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300' 
                     : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700'}"
          >
            <svelte:component 
              this={item.icon} 
              size={20} 
              class="mr-3 flex-shrink-0 {currentPage === item.id 
                      ? 'text-primary-600 dark:text-primary-400' 
                      : 'text-gray-400 group-hover:text-gray-500 dark:text-gray-500 dark:group-hover:text-gray-400'}" 
            />
            {item.label}
          </button>
        </li>
      {/each}
    </ul>
  </nav>

  <!-- Sidebar footer -->
  <div class="absolute bottom-0 left-0 right-0 p-6 border-t border-gray-200 dark:border-gray-700">
    <div class="flex items-center space-x-3">
      <div class="w-8 h-8 bg-gray-300 dark:bg-gray-600 rounded-full flex items-center justify-center">
        <span class="text-xs font-medium text-gray-700 dark:text-gray-300">AI</span>
      </div>
      <div>
        <p class="text-sm font-medium text-gray-900 dark:text-white">
          Система активна
        </p>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          Версия 1.0.0
        </p>
      </div>
    </div>
  </div>
</div> 