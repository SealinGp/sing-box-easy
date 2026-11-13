<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Button from './Button.vue'
import IconButton from './IconButton.vue'
import Skeleton from './Skeleton.vue'
import Popover from '../volt/Popover.vue'

const props = withDefaults(
  defineProps<{
    /** Primary question, e.g. "Delete this rule set?" */
    message: string
    /**
     * The item being acted on, rendered as a prominent monospace chip.
     * This is what tells the user exactly what they are about to destroy.
     */
    target?: string
    /**
     * Optional secondary line — consequences, cascade summaries, warnings.
     * Safe to populate asynchronously after `open` fires.
     */
    details?: string
    /** Blocks confirm and shows a spinner while the parent resolves `details`. */
    loading?: boolean
    confirmLabel?: string
    cancelLabel?: string
    /** Text shown beside the spinner while `loading`. */
    loadingLabel?: string
    tone?: 'danger' | 'primary'
    /** Classes for the trigger button. Overrides the default ghost styling. */
    triggerClass?: string
    /** Which side of the trigger the panel opens toward. */
    align?: 'left' | 'right'
  }>(),
  { tone: 'danger', align: 'right', loading: false },
)

const emit = defineEmits<{ confirm: []; open: [] }>()
const { t } = useI18n()
const panel = ref<InstanceType<typeof Popover> | null>(null)
const open = ref(false)
function close() { panel.value?.hide() }
function confirm() {
  if (props.loading) return
  emit('confirm')
  close()
}
</script>

<template>
  <span class="inline-flex">
    <IconButton
      v-if="tone === 'danger'"
      action="delete"
      :label="`${confirmLabel ?? t('common.delete')}${target ? ` ${target}` : ''}`"
      :aria-expanded="open"
      aria-haspopup="dialog"
      @click="panel?.toggle($event)"
    />
    <button v-else type="button" :class="triggerClass || 'icon-action'"
      :aria-expanded="open" aria-haspopup="dialog" @click="panel?.toggle($event)">
      <slot>{{ confirmLabel ?? t('common.confirm') }}</slot>
    </button>
    <Popover ref="panel" :aria-label="message" @show="open = true; emit('open')" @hide="open = false">
      <div :aria-busy="loading" class="text-left">
        <p class="whitespace-pre-line text-sm text-text-primary">{{ message }}</p>
        <p v-if="target" class="mt-2 max-h-24 overflow-y-auto break-words font-mono text-sm font-medium text-text-primary">{{ target }}</p>
        <div v-if="loading" class="mt-2 space-y-1.5" role="status" :aria-label="loadingLabel ?? t('common.loading')">
          <Skeleton class="h-3 w-full" /><Skeleton class="h-3 w-3/4" />
        </div>
        <p v-else-if="details" class="mt-2 whitespace-pre-line text-xs text-text-secondary">{{ details }}</p>
        <div class="mt-4 flex items-center justify-end gap-2">
          <Button variant="ghost" size="sm" action @click="close">{{ cancelLabel ?? t('common.cancel') }}</Button>
          <IconButton v-if="tone === 'danger'" action="delete" :label="confirmLabel ?? t('common.delete')" :disabled="loading" @click="confirm" />
          <Button v-else size="sm" :disabled="loading" @click="confirm">{{ confirmLabel ?? t('common.confirm') }}</Button>
        </div>
      </div>
    </Popover>
  </span>
</template>
