import { websocket, systemStats, cameras, events, addNotification, updateCameraStatus } from '../stores/index.js'

let ws = null
let reconnectAttempts = 0
let maxReconnectAttempts = 5
let reconnectDelay = 1000

export function connectWebSocket() {
  if (ws && (ws.readyState === WebSocket.CONNECTING || ws.readyState === WebSocket.OPEN)) {
    return
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws`

  try {
    ws = new WebSocket(wsUrl)
    
    ws.onopen = function(event) {
      console.log('WebSocket connected')
      reconnectAttempts = 0
      websocket.set(ws)
      addNotification('Соединение с сервером установлено', 'success', 3000)
    }

    ws.onmessage = function(event) {
      try {
        const data = JSON.parse(event.data)
        handleWebSocketMessage(data)
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    ws.onclose = function(event) {
      console.log('WebSocket disconnected:', event.code, event.reason)
      websocket.set(null)
      
      if (!event.wasClean && reconnectAttempts < maxReconnectAttempts) {
        setTimeout(() => {
          reconnectAttempts++
          console.log(`Reconnection attempt ${reconnectAttempts}/${maxReconnectAttempts}`)
          connectWebSocket()
        }, reconnectDelay * Math.pow(2, reconnectAttempts))
      } else if (reconnectAttempts >= maxReconnectAttempts) {
        addNotification('Не удалось восстановить соединение с сервером', 'error', 10000)
      }
    }

    ws.onerror = function(error) {
      console.error('WebSocket error:', error)
      addNotification('Ошибка соединения с сервером', 'error', 5000)
    }

  } catch (error) {
    console.error('Failed to create WebSocket connection:', error)
    addNotification('Не удалось подключиться к серверу', 'error', 5000)
  }
}

export function disconnectWebSocket() {
  if (ws) {
    ws.close(1000, 'Manual disconnect')
    ws = null
    websocket.set(null)
  }
}

function handleWebSocketMessage(data) {
  switch (data.type) {
    case 'stats_update':
      systemStats.set(data.data)
      break

    case 'camera_status':
      updateCameraStatus(data.camera_id, data.status)
      if (data.status === 'offline') {
        addNotification(`Камера "${data.camera_name}" отключилась`, 'warning', 8000)
      } else if (data.status === 'online') {
        addNotification(`Камера "${data.camera_name}" подключилась`, 'success', 5000)
      }
      break

    case 'new_event':
      // Добавляем новое событие в начало списка
      events.update(items => {
        const newItems = [data.event, ...items]
        // Ограничиваем количество событий в памяти
        return newItems.slice(0, 100)
      })

      // Показываем уведомление
      const eventType = data.event.type === 'motion' ? 'движение' : 'объект'
      addNotification(
        `${data.event.camera_name}: обнаружено ${eventType}`,
        'info',
        6000
      )
      break

    case 'motion_detected':
      addNotification(
        `${data.camera_name}: обнаружено движение`,
        'warning',
        5000
      )
      break

    case 'ai_detection':
      addNotification(
        `${data.camera_name}: обнаружен объект "${data.object_class}" (${Math.round(data.confidence * 100)}%)`,
        'info',
        7000
      )
      break

    case 'system_alert':
      addNotification(data.message, data.level || 'warning', 10000)
      break

    case 'camera_added':
      cameras.update(items => [...items, data.camera])
      addNotification(`Добавлена новая камера: ${data.camera.name}`, 'success', 5000)
      break

    case 'camera_removed':
      cameras.update(items => items.filter(camera => camera.id !== data.camera_id))
      addNotification(`Камера удалена: ${data.camera_name}`, 'info', 5000)
      break

    case 'camera_updated':
      cameras.update(items => 
        items.map(camera => 
          camera.id === data.camera.id ? { ...camera, ...data.camera } : camera
        )
      )
      break

    default:
      console.log('Unknown WebSocket message type:', data.type)
  }
}

// Функция для отправки сообщений через WebSocket
export function sendWebSocketMessage(message) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(message))
    return true
  }
  return false
}

// Проверка состояния соединения
export function isWebSocketConnected() {
  return ws && ws.readyState === WebSocket.OPEN
} 