<script setup lang="ts">
/**
 * Build a "race several resolvers, take the first good answer" group.
 *
 * WHY A DEDICATED FLOW
 * ────────────────────
 * Every other dialog on this page edits ONE rule. This pattern is N+N rules
 * that only work as a complete, ordered set, so the single-rule form cannot
 * express it — and getting it wrong by hand is not a soft failure. sing-box:
 * a `respond` reached with no preceding `evaluate` for its tag makes the
 * request FAIL, rather than falling through to later rules. Three servers is
 * six rules, six tags that must line up exactly, and one ordering constraint
 * that breaks DNS when violated.
 *
 * So the operator names WHAT to match and WHICH servers to race, and the rules
 * are generated. The preview is not decoration either: this writes six rules
 * on one click, and the reader deserves to see them before that happens.
 */
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { Dialog, MultiSelect } from '../volt'
import Button from './Button.vue'
import DNSRuleConditions from './DNSRuleConditions.vue'
import { useDNSStore } from '../stores/dns'
import {
  buildParallelResolveGroup,
  suggestGroupName,
  validateParallelResolveGroup,
  type ParallelConditions,
} from '../utils/parallelResolve'

const props = defineProps<{
  visible: boolean
  /** Rules already in the config, for the tag-collision check. */
  existingRules: Record<string, any>[]
  ruleSetOptions: { value: string; label: string }[]
  saving?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'create', rules: Record<string, any>[]): void
}>()

const dnsStore = useDNSStore()
const { dnsServers } = storeToRefs(dnsStore)

/**
 * Every list is materialised, never undefined: DNSRuleConditions binds Chips
 * fields to these and a Chips box cannot v-model an undefined value. The type
 * says so rather than leaving it to the initialiser.
 */
type FormConditions = Required<ParallelConditions>

const conditions = ref<FormConditions>(emptyConditions())
const servers = ref<string[]>([])
const group = ref('')
const timeout = ref('3s')
/** True once the operator edits the name, so suggestions stop overwriting it. */
const groupTouched = ref(false)

function emptyConditions(): FormConditions {
  return { rule_set: [], domain: [], domain_suffix: [], domain_keyword: [], geosite: [] }
}

const serverOptions = computed(() =>
  (dnsServers.value ?? [])
    .filter((server) => server.tag)
    .map((server) => ({
      value: server.tag as string,
      label: `${server.tag} (${(server as any).type ?? 'udp'})`,
    })),
)

/**
 * The name is suggested from the conditions, not required to be typed.
 *
 * These tags end up in config.json and are referenced by every respond rule in
 * the group, so `deepseek_com_aliyun` is worth a great deal more than
 * `group1_aliyun` to whoever opens the file later. It stops suggesting the
 * moment the operator types, because overwriting a deliberate name is worse
 * than offering none.
 */
watch(
  conditions,
  (value) => {
    if (!groupTouched.value) group.value = suggestGroupName(value)
  },
  { deep: true },
)

const spec = computed(() => ({
  conditions: conditions.value,
  servers: servers.value,
  group: group.value,
  timeout: timeout.value,
}))

/** An i18n key while the spec is unsound, null when it is ready. */
const problem = computed(() => validateParallelResolveGroup(spec.value, props.existingRules))

const preview = computed(() =>
  problem.value ? [] : buildParallelResolveGroup(spec.value),
)

const evaluateCount = computed(() => preview.value.filter((r) => r.action === 'evaluate').length)

const reset = () => {
  conditions.value = emptyConditions()
  servers.value = []
  group.value = ''
  timeout.value = '3s'
  groupTouched.value = false
}

const close = () => {
  emit('update:visible', false)
  reset()
}

const create = () => {
  if (problem.value) return
  emit('create', preview.value)
}

// Reset on open rather than on close, so a failed save leaves the form intact
// to retry instead of silently discarding what was typed.
watch(
  () => props.visible,
  (open) => {
    if (open) reset()
  },
)
</script>

<template>
  <Dialog
    :visible="visible"
    @update:visible="(v: boolean) => { if (!v) close() }"
    :header="$t('dns.rules.parallel.title')"
    modal
    class="w-full max-w-3xl"
  >
    <div class="space-y-4">
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ $t('dns.rules.parallel.desc') }}
      </p>

      <!-- 1. WHAT to match. Same component the single-rule form uses, so the
           two dialogs cannot describe conditions differently. -->
      <section class="space-y-3">
        <div class="flex items-baseline gap-2">
          <span
            class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
            >1</span
          >
          <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
            {{ $t('dns.rules.parallel.whenHeading') }}
          </h4>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ $t('dns.rules.parallel.whenHint') }}
          </span>
        </div>

        <DNSRuleConditions
          v-model:rule-set="conditions.rule_set"
          v-model:domain="conditions.domain"
          v-model:domain-suffix="conditions.domain_suffix"
          v-model:domain-keyword="conditions.domain_keyword"
          v-model:geosite="conditions.geosite"
          :rule-set-options="ruleSetOptions"
        />
      </section>

      <!-- 2. WHICH servers race. -->
      <section class="space-y-3 border-t border-gray-200 dark:border-gray-700 pt-4">
        <div class="flex items-baseline gap-2">
          <span
            class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
            >2</span
          >
          <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
            {{ $t('dns.rules.parallel.serversHeading') }}
          </h4>
        </div>

        <MultiSelect
          v-model="servers"
          :options="serverOptions"
          optionLabel="label"
          optionValue="value"
          filter
          display="chip"
          :placeholder="$t('dns.rules.parallel.serversPlaceholder')"
          class="w-full"
        />

        <div class="grid grid-cols-1 @md:grid-cols-2 gap-3">
          <label class="block">
            <span class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t('dns.rules.parallel.groupName') }}
            </span>
            <input
              v-model="group"
              type="text"
              spellcheck="false"
              class="block w-full px-2.5 py-1.5 text-sm font-mono"
              @input="groupTouched = true"
            />
            <span class="block mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t('dns.rules.parallel.groupNameHint') }}
            </span>
          </label>

          <label class="block">
            <span class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t('dns.rules.parallel.timeout') }}
            </span>
            <input
              v-model="timeout"
              type="text"
              spellcheck="false"
              placeholder="3s"
              class="block w-full px-2.5 py-1.5 text-sm font-mono"
            />
            <span class="block mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t('dns.rules.parallel.timeoutHint') }}
            </span>
          </label>
        </div>
      </section>

      <!-- 3. What will be written. Six rules on one click is too many to
           take on trust. -->
      <section class="space-y-2 border-t border-gray-200 dark:border-gray-700 pt-4">
        <div class="flex items-baseline gap-2">
          <span
            class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
            >3</span
          >
          <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
            {{ $t('dns.rules.parallel.previewHeading') }}
          </h4>
          <span v-if="preview.length" class="text-xs text-gray-500 dark:text-gray-400">
            {{ $t('dns.rules.parallel.previewCount', { count: preview.length, servers: evaluateCount }) }}
          </span>
        </div>

        <p
          v-if="problem"
          class="rounded-control bg-amber-50 dark:bg-amber-950/30 px-3 py-2 text-xs text-amber-800 dark:text-amber-200"
        >
          {{ $t(problem) }}
        </p>

        <template v-else>
          <!-- The ordering is the correctness requirement, so it is what the
               preview shows: every evaluate, then every respond. -->
          <ol class="space-y-1">
            <li
              v-for="(rule, index) in preview"
              :key="index"
              class="rounded-control border px-3 py-1.5 text-xs font-mono break-all"
              :class="
                rule.action === 'evaluate'
                  ? 'border-primary-300 dark:border-primary-800 bg-primary-50/50 dark:bg-primary-950/20'
                  : 'border-emerald-300 dark:border-emerald-800 bg-emerald-50/50 dark:bg-emerald-950/20'
              "
            >
              <span class="text-gray-400 mr-2">{{ index + 1 }}</span>
              <template v-if="rule.action === 'evaluate'">
                {{ $t('dns.rules.parallel.previewEvaluate', { server: rule.server, tag: rule.tag }) }}
              </template>
              <template v-else>
                {{ $t('dns.rules.parallel.previewRespond', { tag: rule.match_response }) }}
              </template>
            </li>
          </ol>

          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ $t('dns.rules.parallel.appendNote') }}
          </p>
        </template>
      </section>
    </div>

    <template #footer>
      <Button @click="close" variant="secondary">{{ $t('common.cancel') }}</Button>
      <Button @click="create" variant="primary" :disabled="!!problem || saving">
        {{ $t('dns.rules.parallel.create') }}
      </Button>
    </template>
  </Dialog>
</template>
