import { ref, readonly } from 'vue'

export type NotificationType = 'success' | 'warning' | 'error' | 'info'
export type NotificationPosition = 'top-left' | 'top' | 'top-right' | 'bottom-left' | 'bottom' | 'bottom-right'

export interface Notification {
  id: string
  type: NotificationType
  title?: string
  message: string
  duration?: number
  position: NotificationPosition
}

interface NotificationOptions {
  type?: NotificationType
  title?: string
  duration?: number
  position?: NotificationPosition
}

const notifications = ref<Notification[]>([])
let notificationId = 0

export function useNotification() {
  const add = (message: string, options: NotificationOptions = {}) => {
    const id = `notification-${++notificationId}`
    const notification: Notification = {
      id,
      type: options.type || 'info',
      title: options.title,
      message,
      duration: options.duration ?? 5000,
      position: options.position || 'top-right',
    }

    notifications.value.push(notification)

    // Auto remove after duration
    if (notification.duration > 0) {
      setTimeout(() => {
        remove(id)
      }, notification.duration)
    }

    return id
  }

  const remove = (id: string) => {
    const index = notifications.value.findIndex(n => n.id === id)
    if (index !== -1) {
      notifications.value.splice(index, 1)
    }
  }

  const success = (message: string, options: Omit<NotificationOptions, 'type'> = {}) => {
    return add(message, { ...options, type: 'success' })
  }

  const error = (message: string, options: Omit<NotificationOptions, 'type'> = {}) => {
    return add(message, { ...options, type: 'error' })
  }

  const warning = (message: string, options: Omit<NotificationOptions, 'type'> = {}) => {
    return add(message, { ...options, type: 'warning' })
  }

  const info = (message: string, options: Omit<NotificationOptions, 'type'> = {}) => {
    return add(message, { ...options, type: 'info' })
  }

  const clear = () => {
    notifications.value = []
  }

  return {
    notifications: readonly(notifications),
    add,
    remove,
    success,
    error,
    warning,
    info,
    clear,
  }
}
