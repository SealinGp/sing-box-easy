<script setup lang="ts">
import type { Subscription } from '../types/api'
import TooltipTarget from './TooltipTarget.vue'
import IconButton from './IconButton.vue'
import PopConfirm from './PopConfirm.vue'
import { CloudArrowDownIcon } from '@heroicons/vue/24/outline'
defineProps<{ subscription: Subscription; busy: boolean; expanded?: boolean }>()
const emit = defineEmits<{ update: []; edit: []; delete: [] }>()
</script>
<template>            <div class="flex items-center justify-end gap-1">
              <TooltipTarget as="button"
                type="button" class="icon-action"
                :aria-busy="busy"
                @click="emit('update')"
                :disabled="busy"
                :text="$t('subscriptions.tooltip.update')"
                :aria-label="$t('subscriptions.update')"
              >
                <!-- Hide the icon while updating: the Button shows only its
                     spinner and is disabled, so the row can't be re-triggered.
                     Shares CloudArrowDownIcon with the header's "update all",
                     which is this same operation across every row. -->
                <CloudArrowDownIcon
                  v-if="!busy"
                  class="h-4 w-4"
                />
                <span v-else class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
                <span v-if="expanded">{{ $t('common.update') }}</span>
              </TooltipTarget>
              <IconButton action="edit" @click="emit('edit')" :label="$t('subscriptions.tooltip.edit')" />
              <!-- Keep the confirmation attached to the named subscription. -->
              <PopConfirm
                :message="$t('subscriptions.del.confirmHeading')"
                :target="subscription.name"
                :details="$t('subscriptions.del.confirmDesc')"
                :confirm-label="$t('common.delete')"
                tone="danger"
                align="right"
                @confirm="emit('delete')"
              />
            </div>
          </template>
