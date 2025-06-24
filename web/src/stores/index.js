import { writable } from 'svelte/store'

// Системная статистика
export const systemStats = writable({
  cameras_total: 0,
  cameras_online: 0,
  events_today: 0,
  events_total: 0,
  system_uptime: 0
})

// Камеры
export const cameras = writable([])

// События
export const events = writable([])

// WebSocket соединение
export const websocket = writable(null)

// Уведомления
export const notifications = writable([])

// Настройки UI
export const uiSettings = writable({
  darkMode: false,
  sidebarCollapsed: false,
  gridCols: 2
})

// Активные алерты
export const alerts = writable([])

// Функции для работы с уведомлениями
let notificationId = 0

export function addNotification(message, type = 'info', duration = 5000) {
  const id = ++notificationId
  const notification = { id, message, type, timestamp: Date.now() }
  
  notifications.update(items => [...items, notification])
  
  if (duration > 0) {
    setTimeout(() => {
      removeNotification(id)
    }, duration)
  }
  
  return id
}

export function removeNotification(id) {
  notifications.update(items => items.filter(item => item.id !== id))
}

// Функции для работы с алертами
export function addAlert(alert) {
  alerts.update(items => {
    // Удаляем дубликаты по camera_id
    const filtered = items.filter(item => item.camera_id !== alert.camera_id)
    return [...filtered, { ...alert, id: Date.now() }]
  })
}

export function removeAlert(id) {
  alerts.update(items => items.filter(item => item.id !== id))
}

// Функция для обновления статуса камеры
export function updateCameraStatus(cameraId, status) {
  cameras.update(items => 
    items.map(camera => 
      camera.id === cameraId 
        ? { ...camera, status, last_seen: new Date().toISOString() }
        : camera
    )
  )
} 