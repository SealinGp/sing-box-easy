<script setup lang="ts">
/**
 * Editor for an inbound's `users` array.
 *
 * Every inbound type spells a user differently, and the differences matter on
 * the wire — mixed/http/socks/naive carry sing's `auth.User` (`username` +
 * `password`), VMess wants `uuid` + `alterId`, VLESS wants `uuid` + `flow`,
 * TUIC wants all three. The shape comes in as `fields` from the schema rather
 * than being branched on here, so adding a type is a data change.
 *
 * Before this, `users` was editable for exactly two types (shadowsocks and
 * vmess) through hand-written blocks that indexed `users[0]` directly — so a
 * multi-user inbound could not be edited at all, and every other type that
 * takes users had no control whatsoever. Creating a trojan inbound through the
 * old modal produced one with no credentials.
 */
import { useI18n } from 'vue-i18n'
import { PlusIcon, TrashIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'
import Input from './Input.vue'
import Button from './Button.vue'
import { Select } from '../volt'
import { humanizeFieldName } from '../utils/fieldLabels'
import { VLESS_FLOW_OPTIONS, type UserFieldSpec } from '../schemas/inboundFields'
import { generateVmessUUID, generateSecret } from '../utils/credentials'

const props = defineProps<{
  modelValue?: Record<string, unknown>[]
  fields: UserFieldSpec[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>[]]
}>()

const { t, te } = useI18n()

function label(key: string): string {
  const path = `inbounds.form.userFields.${key}`
  return te(path) ? t(path) : humanizeFieldName(key)
}

const users = () => props.modelValue ?? []

/** Immutable throughout: every edit rebuilds the array rather than patching in place. */
function replace(index: number, patch: Record<string, unknown>) {
  emit(
    'update:modelValue',
    users().map((user, i) => (i === index ? { ...user, ...patch } : user)),
  )
}

function addUser() {
  const blank: Record<string, unknown> = {}
  for (const field of props.fields) {
    // Seed the identity field so a fresh entry is immediately usable rather
    // than silently invalid — a UUID nobody can guess, a password nobody
    // shares. Names stay empty; they are labels, not credentials.
    if (field.identity && field.key === 'uuid') blank[field.key] = generateVmessUUID()
    else if (field.identity && field.control === 'password') blank[field.key] = generateSecret()
    else if (field.control === 'number') blank[field.key] = 0
    else blank[field.key] = ''
  }
  emit('update:modelValue', [...users(), blank])
}

function removeUser(index: number) {
  emit(
    'update:modelValue',
    users().filter((_, i) => i !== index),
  )
}

function regenerate(index: number, field: UserFieldSpec) {
  replace(index, { [field.key]: field.key === 'uuid' ? generateVmessUUID() : generateSecret() })
}

/** A field worth offering a "generate" button for — credentials, not labels. */
function isGeneratable(field: UserFieldSpec) {
  return field.key === 'uuid' || field.control === 'password'
}

function flowOptions() {
  return VLESS_FLOW_OPTIONS.map((value) => ({
    value,
    label: value === '' ? t('inbounds.form.flowNone') : value,
  }))
}
</script>

<template>
  <div class="space-y-2">
    <div
      v-for="(user, index) in users()"
      :key="index"
      class="rounded-control border border-gray-200 dark:border-gray-700 p-2"
    >
      <div class="flex items-start gap-2">
        <div class="flex-1 grid grid-cols-2 gap-2">
          <div v-for="field in fields" :key="field.key">
            <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-0.5">
              {{ label(field.key) }}
            </label>

            <Select
              v-if="field.control === 'select'"
              class="w-full"
              :modelValue="user[field.key] ?? ''"
              :options="flowOptions()"
              optionLabel="label"
              optionValue="value"
              @update:modelValue="(v: unknown) => replace(index, { [field.key]: v })"
            />

            <div v-else class="flex gap-1">
              <Input
                :modelValue="(user[field.key] as string | number | undefined) ?? ''"
                :type="field.control === 'number' ? 'number' : 'text'"
                @update:modelValue="
                  (v: string | number) =>
                    replace(index, { [field.key]: field.control === 'number' ? Number(v) : v })
                "
              />
              <Button
                v-if="isGeneratable(field)"
                type="button"
                variant="secondary"
                size="sm"
                action
                class="shrink-0"
                :title="t('inbounds.form.generate')"
                @click="regenerate(index, field)"
              >
                <ArrowPathIcon class="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>

        <Button
          type="button"
          variant="ghost"
          size="sm"
          action
          class="text-red-600 dark:text-red-400 shrink-0"
          :title="t('common.delete')"
          @click="removeUser(index)"
        >
          <TrashIcon class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <button
      type="button"
      class="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-control border border-dashed border-gray-300 dark:border-gray-600 text-xs text-gray-500 dark:text-gray-400 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
      @click="addUser"
    >
      <PlusIcon class="h-3.5 w-3.5" />
      {{ t('inbounds.form.addUser') }}
    </button>
  </div>
</template>
