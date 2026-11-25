<script setup lang="ts">
import { computed, onMounted, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import * as monaco from 'monaco-editor'
import Button from '../Button.vue'
import Input from '../Input.vue'
import Textarea from '../Textarea.vue'
import Select from '../../volt/Select.vue'
import Dialog from '../../volt/Dialog.vue'
import MonacoEditor from '../MonacoEditor.vue'
import MonacoDiffEditor from '../MonacoDiffEditor.vue'
import { fields } from './fields'
import { readLocalRuleFile, saveLocalRuleFile, type LocalRuleFile, type RuleSource } from '../../services/localRuleFile'

const props = defineProps<{ initialTag: string; existing: boolean }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t, locale } = useI18n()
const tag = ref(props.initialTag)
const loaded = ref<LocalRuleFile>()
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const mode = ref('form')
const source = ref<RuleSource>({ version: 3, rules: [{ domain: [], domain_suffix: [] }] })
const raw = ref('')
const pending = ref<RuleSource>()
const showDiff = ref(false)
const nextField = ref<string | null>(null)
const modelUri = 'inmemory://local-rule-set/' + useId() + '.json'
const title = (f: typeof fields[number]) => locale.value.startsWith('zh') ? f.label : f.key
const options = computed(() => fields.map(f => ({ ...f, label: title(f) })))
const path = computed(() => {
  if (props.existing) return loaded.value?.path ?? ''
  const name = tag.value.trim().replace(/[<>:"/\\|?*\u0000-\u001f]/g, '_').replace(/[. ]+$/g, '')
  return (loaded.value?.directory ?? '') + '/' + (name || '…') + '.json'
})
const serialize = (v: unknown) => JSON.stringify(v, null, 2)
const message = (e: unknown) => e instanceof Error ? e.message : String(e)
function formSupported(data: RuleSource) {
  return data.rules.every(r => r && typeof r === 'object' && !Array.isArray(r) &&
    Object.entries(r).every(([key, value]) => fields.some(f => f.key === key) && Array.isArray(value) &&
      value.every(v => typeof v === 'string' || typeof v === 'number')))
}
onMounted(async () => {
  try {
    loaded.value = await readLocalRuleFile(props.existing ? props.initialTag : '')
    if (props.existing) {
      source.value = JSON.parse(JSON.stringify(loaded.value.content))
      if (!formSupported(source.value)) mode.value = 'json'
    }
    raw.value = serialize(source.value)
  } catch (e) { error.value = message(e) }
  finally { loading.value = false }
})
const defaults = monaco.json.jsonDefaults
const oldSchemas = defaults.diagnosticsOptions.schemas ?? []
defaults.setDiagnosticsOptions({
  ...defaults.diagnosticsOptions, validate: true,
  schemas: [...oldSchemas.filter(s => s.uri !== 'inmemory://local-rule-set/schema'), {
    uri: 'inmemory://local-rule-set/schema', fileMatch: ['inmemory://local-rule-set/*'],
    schema: { type: 'object', required: ['version', 'rules'], properties: {
      version: { type: 'integer', minimum: 1 },
      rules: { type: 'array', items: { type: 'object', properties: {
        ...Object.fromEntries(fields.map(f => [f.key, { description: title(f), type: ['array', f.numeric ? 'integer' : 'string'],
          items: { type: f.numeric ? 'integer' : 'string' }, default: [] }])),
        invert: { type: 'boolean' }, type: { enum: ['logical'] }, mode: { enum: ['and', 'or'] }, rules: { type: 'array' },
      } } },
    } },
  }],
})
function changeMode(value: string) {
  if (value === mode.value) return
  error.value = ''
  if (value === 'json') raw.value = serialize(source.value)
  else {
    try {
      const data = JSON.parse(raw.value)
      if (!Array.isArray(data?.rules) || !formSupported(data)) throw new Error(t('localRules.advanced'))
      source.value = data
    } catch (e) { error.value = message(e); return }
  }
  mode.value = value
}
function update(rule: Record<string, unknown>, key: string, value: string) {
  const parts = value.split(/\r?\n/).map(s => s.trim()).filter(Boolean)
  rule[key] = fields.find(f => f.key === key)?.numeric ? parts.map(Number) : parts
}
function preview() {
  error.value = ''
  try {
    if (!tag.value.trim()) throw new Error(t('localRules.nameRequired'))
    const data = mode.value === 'json' ? JSON.parse(raw.value) : JSON.parse(serialize(source.value))
    if (!Number.isInteger(data?.version) || !Array.isArray(data?.rules)) throw new Error(t('localRules.invalid'))
    if (mode.value === 'form') {
      data.rules = data.rules.map((r: Record<string, unknown>) =>
        Object.fromEntries(Object.entries(r).filter(([, v]) => !(Array.isArray(v) && !v.length))))
    }
    if (!data.rules.length || data.rules.some((r: unknown) => !r || typeof r !== 'object' || Array.isArray(r) || !Object.keys(r).length)) throw new Error(t('localRules.empty'))
    pending.value = data
    showDiff.value = true
  } catch (e) { error.value = message(e) }
}
async function save() {
  if (!loaded.value || !pending.value || busy.value) return
  busy.value = true
  error.value = ''
  try {
    await saveLocalRuleFile(tag.value.trim(), props.existing, loaded.value.revision, pending.value)
    emit('saved')
  } catch (e) { error.value = message(e) }
  finally { busy.value = false }
}
</script>

<template>
  <Dialog :visible="true" modal class="w-full max-w-6xl" :header="t('localRules.title')"
    :closable="!busy" :close-on-escape="!busy" @update:visible="(value: boolean) => { if (!value && !busy) emit('close') }">
    <p v-if="loading" role="status">{{ t('common.loading') }}</p>
    <div v-else-if="loaded" class="space-y-5">
      <div class="grid gap-5 md:grid-cols-2">
        <Input v-model="tag" :disabled="existing || busy" :label="t('localRules.name')" />
        <div class="space-y-2"><p class="text-sm font-medium">{{ t('localRules.path') }}</p><p class="break-all font-mono text-xs text-text-secondary">{{ path }}</p></div>
      </div>
      <template v-if="!showDiff">
        <div role="group" :aria-label="t('localRules.editMode')" class="flex gap-2">
          <Button :variant="mode === 'form' ? 'primary' : 'ghost'" :aria-pressed="mode === 'form'" @click="changeMode('form')">{{ t('localRules.visual') }}</Button>
          <Button :variant="mode === 'json' ? 'primary' : 'ghost'" :aria-pressed="mode === 'json'" @click="changeMode('json')">JSON</Button>
        </div>
        <div v-if="mode === 'form'" class="space-y-4">
          <article v-for="(rule, index) in source.rules" :key="index" class="space-y-4 rounded-control border border-border p-4">
            <div class="flex justify-between"><h3 class="text-sm font-semibold">{{ t('localRules.rule', { n: index + 1 }) }}</h3>
              <Button v-if="source.rules.length > 1" variant="ghost" @click="source.rules.splice(index, 1)">{{ t('common.delete') }}</Button></div>
            <div class="grid gap-4 md:grid-cols-2">
              <div v-for="key in Object.keys(rule)" :key="key">
                <div class="mb-1 flex justify-between"><span class="text-xs text-text-secondary">{{ key }}</span>
                  <button type="button" class="px-2 text-xs underline" @click="delete rule[key]">{{ t('localRules.remove') }}</button></div>
                <Textarea :model-value="(rule[key] as unknown[]).join('\n')" @update:model-value="value => update(rule, key, String(value))"
                  :label="options.find(f => f.key === key)?.label ?? key" :placeholder="options.find(f => f.key === key)?.example" :rows="4" />
              </div>
            </div>
            <Select :model-value="nextField" :options="options.filter(f => !(f.key in rule))" option-label="label" option-value="key"
              filter :placeholder="t('localRules.addField')" :aria-label="t('localRules.addField')" class="w-full md:w-72"
              @update:model-value="(value: string) => { rule[value] = []; nextField = null }" />
          </article>
          <Button variant="ghost" @click="source.rules.push({ domain_suffix: [] })">{{ t('localRules.addRule') }}</Button>
        </div>
        <div v-else class="h-[52vh]"><MonacoEditor v-model="raw" :model-uri="modelUri" :options="{ minimap: { enabled: false } }" class="h-full" /></div>
      </template>
      <template v-else>
        <div class="flex justify-between text-xs"><span>{{ t('managed.before') }}</span><span>{{ t('managed.after') }}</span></div>
        <div class="h-[55vh]"><MonacoDiffEditor :original="serialize(loaded.content)" :modified="serialize(pending)" /></div>
        <p class="text-xs text-text-secondary">{{ t('localRules.validation') }}</p>
      </template>
    </div>
    <p v-if="error" role="alert" class="mt-3 break-words text-sm text-red-600">{{ error }}</p>
    <template #footer>
      <Button variant="ghost" :disabled="busy" @click="showDiff ? (showDiff = false) : emit('close')">{{ t(showDiff ? 'common.back' : 'common.cancel') }}</Button>
      <Button v-if="loaded" :loading="busy" @click="showDiff ? save() : preview()">{{ t(showDiff ? 'common.save' : 'localRules.preview') }}</Button>
    </template>
  </Dialog>
</template>
