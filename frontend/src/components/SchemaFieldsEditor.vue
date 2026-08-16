<script setup lang="ts">
/**
 * The type-aware body of a schema-driven dialog.
 *
 * Shared by the inbound and DNS server forms; it is handed an already-resolved
 * field list and never learns which domain it is drawing. Between them it
 * replaces ~250 lines of hand-written `v-if="type === '...'"` template that
 * covered 3 of 16 inbound types and left DNS `local` servers with no editable
 * field at all.
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
import SchemaFieldControl from './SchemaFieldControl.vue'
import { useOptionalFields } from '../composables/useMatcherFields'
import { humanizeFieldName } from '../utils/fieldLabels'
import {
  isDeprecatedIn,
  isFieldFilled,
  isRetired,
  withoutField,
  type ResolvedField,
} from '../schemas/optionSchema'
import { useSingBoxVersion } from '../composables/useSingBoxVersion'
import type { UserFieldSpec } from '../schemas/inboundFields'

const props = defineProps<{
  /** Already resolved by the domain's schema, so this stays domain-agnostic. */
  fields: ResolvedField[]
  /** Sub-field shape for a `users` control, when the domain has one. */
  userFields?: UserFieldSpec[]
  /**
   * Shown when a type has no core or typical field at all. DNS `local` is the
   * real case: it resolves through the host resolver and owns nothing but
   * dialer options, so without this the dialog would look broken.
   */
  emptyHint?: string
}>()

const model = defineModel<Record<string, unknown>>({ required: true })

const { t, te } = useI18n()

// The installed binary, not the pinned library — see useSingBoxVersion.
const { singBoxVersion } = useSingBoxVersion()

/** Removed by the sing-box actually installed: offering it is a failed save. */
const retired = (field: ResolvedField) => isRetired(field, singBoxVersion.value)

const fields = computed(() => props.fields)

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
    // Deprecated but still accepted by the installed binary: offered, flagged.
    deprecated: hidden.filter((f) => f.deprecated && !retired(f)),
    // Removed by the installed binary. NOT offered — adding one makes the save
    // fail with an upstream decode error for something the UI suggested.
    // Counted so the row can say what is being withheld and why.
    retired: hidden.filter((f) => retired(f)),
  }
})

function label(field: ResolvedField): string {
  return te(field.labelKey) ? t(field.labelKey) : humanizeFieldName(field.key)
}

function hint(field: ResolvedField): string | undefined {
  // A retired field still renders when a loaded config uses it — hiding it
  // would mean the operator could not see or clear the thing breaking their
  // config — but it says plainly that this binary will reject it.
  if (retired(field)) {
    return t('schema.field.retired', {
      removed: field.removed,
      version: singBoxVersion.value,
    })
  }
  if (isDeprecatedIn(field, singBoxVersion.value)) {
    return t('schema.field.deprecated', { since: field.since })
  }
  if (field.deprecated) return t('schema.field.deprecatedUnversioned')
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
      <SchemaFieldControl
        :field="field"
        :value="model[field.key]"
        :record="model"
        :user-fields="props.userFields"
        @change="(v: unknown) => setField(field.key, v)"
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
      <SchemaFieldControl
        :field="field"
        :value="model[field.key]"
        :record="model"
        :user-fields="props.userFields"
        @change="(v: unknown) => setField(field.key, v)"
      />
    </LabeledField>

    <p
      v-if="emptyHint && !coreFields.length && !shownOptional.length"
      class="text-xs text-gray-500 dark:text-gray-400"
    >
      {{ emptyHint }}
    </p>

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

    <!--
      Removed by the installed sing-box. Stated rather than silently dropped:
      a field vanishing with no explanation reads as a missing feature, and the
      operator would go looking for it in config.json instead.
    -->
    <p
      v-if="hiddenFields.retired.length"
      class="text-xs text-gray-500 dark:text-gray-400"
    >
      {{
        t('schema.field.retiredHidden', {
          count: hiddenFields.retired.length,
          version: singBoxVersion,
        })
      }}
    </p>

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
