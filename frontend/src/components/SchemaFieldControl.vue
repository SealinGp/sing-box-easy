<script setup lang="ts">
/**
 * Renders ONE schema field, picking the control from its resolved entry.
 *
 * Domain-agnostic: it is given a ResolvedField and a value, and never learns
 * whether it is drawing an inbound or a DNS server. The `control` value is
 * already decided by the schema, so this never branches on a field's name —
 * the one exception being the composite controls (`users`, `hosts`), which
 * need a dedicated editor rather than a text box.
 *
 * Split out of SchemaFieldsEditor so orchestration (which fields are visible,
 * which can be added or removed) stays readable next to rendering (which
 * control a `duration` gets).
 */
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useOutboundsStore } from '../stores/outbounds'
import Input from './Input.vue'
import ChipsField from './ChipsField.vue'
import JsonField from './JsonField.vue'
import UsersEditor from './UsersEditor.vue'
import HostsEditor from './HostsEditor.vue'
import { Select } from '../volt'
import type { ResolvedField } from '../schemas/optionSchema'
import type { UserFieldSpec } from '../schemas/inboundFields'

const props = defineProps<{
  field: ResolvedField
  value: unknown
  userFields?: UserFieldSpec[]
  disabled?: boolean
}>()

const emit = defineEmits<{ change: [value: unknown] }>()

const { t } = useI18n()

/**
 * Outbound tags for a `detour` picker. Fetched lazily and only when a detour
 * field is actually rendered, so a form with no detour costs no request.
 * A tag must be picked rather than typed: naming an outbound that does not
 * exist makes sing-box reject the whole config, and the typo is invisible
 * until the next validate.
 */
const outboundsStore = useOutboundsStore()
const { outbounds } = storeToRefs(outboundsStore)

onMounted(() => {
  if (props.field.control === 'outbound' && outbounds.value.length === 0) {
    outboundsStore.fetchOutbounds().catch(() => {
      // Non-fatal: the picker falls back to whatever the record already holds.
    })
  }
})

const outboundOptions = computed(() => {
  const options = [{ value: '', label: t('common.direct') }]
  for (const outbound of outbounds.value) {
    if (outbound.tag) options.push({ value: outbound.tag, label: outbound.tag })
  }
  // A detour naming an outbound that was since renamed would otherwise vanish
  // from the picker while still living in the config, and the next save would
  // look like the operator had cleared it. Surface it instead.
  const current = props.value
  if (typeof current === 'string' && current && !options.some((o) => o.value === current)) {
    options.push({ value: current, label: t('common.missingTag', { tag: current }) })
  }
  return options
})

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

  <HostsEditor
    v-else-if="field.control === 'hosts'"
    :modelValue="(value as Record<string, string | string[]> | undefined)"
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
    v-else-if="field.control === 'outbound'"
    class="w-full"
    :modelValue="value ?? ''"
    :options="outboundOptions"
    optionLabel="label"
    optionValue="value"
    :disabled="disabled"
    @update:modelValue="(v: unknown) => emit('change', v === '' ? undefined : v)"
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
