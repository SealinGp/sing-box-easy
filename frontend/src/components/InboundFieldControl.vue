<script setup lang="ts">
/**
 * Renders ONE inbound field, picking the control from its resolved schema entry.
 *
 * Split out of InboundFieldsEditor so the orchestration (which fields are
 * visible, which can be added or removed) stays readable next to the rendering
 * (which control a `duration` gets). The `control` value is already decided by
 * the schema — this component never branches on the field's name.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Input from './Input.vue'
import ChipsField from './ChipsField.vue'
import JsonField from './JsonField.vue'
import UsersEditor from './UsersEditor.vue'
import { Select } from '../volt'
import type { ResolvedField, UserFieldSpec } from '../schemas/inboundFields'

const props = defineProps<{
  field: ResolvedField
  value: unknown
  userFields?: UserFieldSpec[]
  disabled?: boolean
}>()

const emit = defineEmits<{ change: [value: unknown] }>()

const { t } = useI18n()

const selectOptions = computed(() =>
  (props.field.options ?? []).map((value) => ({ value, label: value })),
)

/**
 * Chips models `string[]`, but several list fields are numeric on the wire
 * (`include_uid`, `exclude_uid`). Round-trip through strings and convert back
 * only for entries that are entirely digits, so a range like "1000:2000" — which
 * sing-box accepts for the *_range variants — survives as a string.
 */
const chipsValue = computed(() =>
  Array.isArray(props.value) ? props.value.map((entry) => String(entry)) : [],
)

function onChips(next: string[]) {
  if (props.field.item === 'number') {
    emit(
      'change',
      next.map((entry) => (/^\d+$/.test(entry) ? Number(entry) : entry)),
    )
    return
  }
  emit('change', next)
}

function onNumber(raw: string | number) {
  // An emptied number input must clear the key, not write NaN or 0 — `0` is a
  // meaningful value for several sing-box fields (a listen_port of 0 asks it to
  // pick one), so it cannot double as "unset".
  if (raw === '' || raw === null || raw === undefined) {
    emit('change', undefined)
    return
  }
  const parsed = Number(raw)
  emit('change', Number.isNaN(parsed) ? undefined : parsed)
}
</script>

<template>
  <UsersEditor
    v-if="field.control === 'users'"
    :modelValue="(value as Record<string, unknown>[] | undefined)"
    :fields="userFields ?? []"
    @update:modelValue="(v) => emit('change', v)"
  />

  <JsonField
    v-else-if="field.control === 'json'"
    :modelValue="value"
    :disabled="disabled"
    @update:modelValue="(v) => emit('change', v)"
  />

  <ChipsField
    v-else-if="field.control === 'chips'"
    :modelValue="chipsValue"
    :placeholder="field.placeholder"
    :disabled="disabled"
    @update:modelValue="onChips"
  />

  <Select
    v-else-if="field.control === 'select'"
    class="w-full"
    :modelValue="value ?? ''"
    :options="selectOptions"
    optionLabel="label"
    optionValue="value"
    :disabled="disabled"
    @update:modelValue="(v: unknown) => emit('change', v)"
  />

  <label
    v-else-if="field.control === 'switch'"
    class="flex items-center gap-2 cursor-pointer select-none"
  >
    <input
      type="checkbox"
      :checked="value === true"
      :disabled="disabled"
      class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      @change="emit('change', ($event.target as HTMLInputElement).checked)"
    />
    <span class="text-sm text-gray-700 dark:text-gray-300">
      {{ t('inbounds.form.enabled') }}
    </span>
  </label>

  <Input
    v-else-if="field.control === 'number'"
    :modelValue="(value as number | undefined) ?? ''"
    type="number"
    :placeholder="field.placeholder"
    :disabled="disabled"
    @update:modelValue="onNumber"
  />

  <Input
    v-else
    :modelValue="(value as string | undefined) ?? ''"
    :type="field.control === 'password' ? 'password' : 'text'"
    :placeholder="field.placeholder ?? (field.kind === 'duration' ? '5m' : undefined)"
    :disabled="disabled"
    @update:modelValue="(v) => emit('change', v === '' ? undefined : v)"
  />
</template>
