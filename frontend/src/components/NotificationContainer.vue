<script setup lang="ts">
import { computed } from 'vue'
import { TransitionGroup } from 'vue'
import Notification from './Notification.vue'
import { useNotification, type NotificationPosition } from '../composables/useNotification'

const { notifications, remove } = useNotification()

// Group notifications by position
const notificationsByPosition = computed(() => {
  const grouped: Record<NotificationPosition, Array<typeof notifications.value[number]>> = {
    'top-left': [],
    'top': [],
    'top-right': [],
    'bottom-left': [],
    'bottom': [],
    'bottom-right': [],
  }

  notifications.value.forEach(notification => {
    grouped[notification.position].push(notification)
  })

  return grouped
})

const positionClasses: Record<NotificationPosition, string> = {
  'top-left': 'top-4 left-4 items-start',
  'top': 'top-4 left-1/2 -translate-x-1/2 items-center',
  'top-right': 'top-4 right-4 items-end',
  'bottom-left': 'bottom-4 left-4 items-start',
  'bottom': 'bottom-4 left-1/2 -translate-x-1/2 items-center',
  'bottom-right': 'bottom-4 right-4 items-end',
}
</script>

<template>
  <div class="notification-container">
    <div
      v-for="(position, key) in positionClasses"
      :key="key"
      :class="['fixed z-50 flex flex-col gap-3 pointer-events-none', position]"
    >
      <TransitionGroup
        name="notification"
        tag="div"
        class="flex flex-col gap-3"
      >
        <div
          v-for="notification in notificationsByPosition[key as NotificationPosition]"
          :key="notification.id"
          class="pointer-events-auto"
        >
          <Notification
            :type="notification.type"
            :title="notification.title"
            :message="notification.message"
            @close="remove(notification.id)"
          />
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>

<style scoped>
/* Slide from right for right-side positions */
.notification-enter-active,
.notification-leave-active {
  transition: all 0.3s ease;
}

.notification-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.notification-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

/* Slide from left for left-side positions */
:deep(.items-start) .notification-enter-from {
  transform: translateX(-100%);
}

:deep(.items-start) .notification-leave-to {
  transform: translateX(-100%);
}

/* Slide from top for center top position */
:deep(.top-4.items-center) .notification-enter-from {
  transform: translate(-50%, -100%);
}

:deep(.top-4.items-center) .notification-leave-to {
  transform: translate(-50%, -100%);
}

/* Slide from bottom for center bottom position */
:deep(.bottom-4.items-center) .notification-enter-from {
  transform: translate(-50%, 100%);
}

:deep(.bottom-4.items-center) .notification-leave-to {
  transform: translate(-50%, 100%);
}

/* Smooth height transitions for list */
.notification-move {
  transition: transform 0.3s ease;
}
</style>
