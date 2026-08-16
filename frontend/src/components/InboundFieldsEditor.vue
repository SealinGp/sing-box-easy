<script setup lang="ts">
/**
 * The type-aware body of the inbound dialog.
 *
 * Replaces ~135 lines of hand-written `v-if="type === 'shadowsocks'"` template
 * that covered 3 of 16 inbound types. Every other type — trojan, vless, tuic,
 * hysteria2, naive, shadowtls and the rest — rendered nothing beyond tag, listen
 * address and port, so a trojan inbound created through this panel had no
 * credentials and could authenticate nobody.
 *
 * WHAT DECIDES VISIBILITY
 * ───────────────────────
 *   core      always rendered, no remove control
 *   typical   seeded as shown on open, removable while empty
 *   advanced  hidden until the operator adds it, OR until a loaded config turns
 *             out to use it — a field holding a value is ALWAYS rendered, so
 *             opening an existing inbound can never hide part of it
 *
 * The add/remove mechanics come from `useOptionalFields`, the same composable
 * behind the route-rule and DNS-rule condition forms. Reusing it keeps the three
 * "add a field" rows behaving identically, and it already encodes the rule that
 * a field holding a value is not removable — removing it would discard the value
 * with no undo.
 *
 * RESET ON TYPE CHANGE is the caller's job, via `:key="type"`. Remounting drops
 * the `added` set, so switching shadowsocks → trojan does not leave shadowsocks'
 * fields showing. Same trick `RoutingRules.vue` uses with `matchersKey`.
 */
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { PlusCircleIcon, ExclamationTriangleIcon } from '@heroicons/vue/24/outline'
import LabeledField from './LabeledField.vue'
import InboundFieldControl from './InboundFieldControl.vue'
import { useOptionalFields } from '../composables/useMatcherFields'
import { humanizeFieldName } from '../utils/fieldLabels'
import {
  isFieldFilled,
  resolveInboundFields,
  withoutField,
  USER_FIELDS,
  type InboundTypeName,
  type ResolvedField,
} from '../schemas/inboundFields'

const props = defineProps<{ type: InboundTypeName }>()

const model = defineModel<Record<string, unknown>>({ required: true })

const { t, te } = useI18n()

const fields = computed(() => resolveInboundFields(props.type))

const coreFields = computed(() => fields.value.filter((f) => f.tier === 'core'))
/** Everything the operator may add or remove: typical + advanced. */
const optionalFields = computed(() => fields.value.filter((f) => f.tier !== 'core'))

const byKey = computed(() => new Map(fields.value.map((f) => [f.key, f])))

// Plain array, not a computed: the field list is a pure function of `type`, and
// the component remounts whenever `type` changes (see the :key note above), so
// it cannot go stale.
const optionalKeys = optionalFields.value.map((f) => f.key)

const optional = useOptionalFields(optionalKeys, (key: string) =>
  isFieldFilled(model.value[key]),
)

// Typical fields are SEEDED rather than force-shown. Forcing them would make
// them permanently un-removable, since removal is only offered for a field the
// operator opted into; seeding gives the same default layout while leaving an
// unwanted one dismissible.
onMounted(() => {
  for (const field of optionalFields.value) {
    if (field.tier === 'typical') optional.add(field.key)
  }
})

const shownOptional = computed(() => optionalFields.value.filter((f) => optional.isShown(f.key)))

/**
 * Hidden fields, split so the "add" row leads with what is plausible.
 * Deprecated fields are offered last and flagged — they are still editable
 * because existing configs use them, but nothing should steer someone new
 * toward `sniff` when it moved to route rules in 1.12.
 */
const hiddenFields = computed(() => {
  const hidden = optional.hidden.value
    .map((key) => byKey.value.get(key))
    .filter((f): f is ResolvedField => !!f)
  return {
    current: hidden.filter((f) => !f.deprecated),
    deprecated: hidden.filter((f) => f.deprecated),
  }
})

const userFields = computed(() => USER_FIELDS[props.type])

function label(field: ResolvedField): string {
  return te(field.labelKey) ? t(field.labelKey) : humanizeFieldName(field.key)
}

function hint(field: ResolvedField): string | undefined {
  if (field.deprecated) return t('inbounds.form.deprecatedHint')
  return field.hintKey && te(field.hintKey) ? t(field.hintKey) : undefined
}

/** Immutable update — replace the object rather than patching it in place. */
function setField(key: string, value: unknown) {
  if (value === undefined) {
    model.value = withoutField(model.value, key)
    return
  }
  model.value = { ...model.value, [key]: value }
}

/**
 * Removing a field DELETES the key rather than blanking it. Writing `""` or
 * `false` back would persist the field into config.json as an explicit setting,
 * which is not the same as absent — and would make the field permanently
 * un-removable next time, since a written value counts as filled.
 */
function removeField(key: string) {
  optional.remove(key)
  model.value = withoutField(model.value, key)
}
</script>

<template>
  <div class="space-y-3">
    <!-- Core: the type cannot work without these, so no remove control. -->
    <LabeledField
      v-for="field in coreFields"
      :key="field.key"
      :label="label(field)"
      :hint="hint(field)"
    >
      <InboundFieldControl
        :field="field"
        :value="model[field.key]"
        :user-fields="userFields"
        @change="(v) => setField(field.key, v)"
      />
    </LabeledField>

    <LabeledField
      v-for="field in shownOptional"
      :key="field.key"
      :label="label(field)"
      :hint="hint(field)"
      :removable="optional.isRemovable(field.key)"
      @remove="removeField(field.key)"
    >
      <InboundFieldControl
        :field="field"
        :value="model[field.key]"
        :user-fields="userFields"
        @change="(v) => setField(field.key, v)"
      />
    </LabeledField>

    <!-- Add-field rows: also the discovery mechanism for what this type accepts. -->
    <div
      v-if="hiddenFields.current.length"
      class="flex flex-wrap items-center gap-1.5 border-t border-gray-200 dark:border-gray-700 pt-3"
    >
      <span class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('inbounds.form.addField') }}
      </span>
      <button
        v-for="field in hiddenFields.current"
        :key="field.key"
        type="button"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-pill border border-dashed border-gray-300 dark:border-gray-600 text-xs text-gray-600 dark:text-gray-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
        @click="optional.add(field.key)"
      >
        <PlusCircleIcon class="h-3 w-3" />
        {{ label(field) }}
      </button>
    </div>

    <div v-if="hiddenFields.deprecated.length" class="flex flex-wrap items-center gap-1.5">
      <span class="inline-flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
        <ExclamationTriangleIcon class="h-3.5 w-3.5" />
        {{ t('inbounds.form.addDeprecatedField') }}
      </span>
      <button
        v-for="field in hiddenFields.deprecated"
        :key="field.key"
        type="button"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-pill border border-dashed border-amber-300 dark:border-amber-700 text-xs text-amber-700 dark:text-amber-400 hover:border-amber-500 transition-colors"
        @click="optional.add(field.key)"
      >
        {{ label(field) }}
      </button>
    </div>
  </div>
</template>
