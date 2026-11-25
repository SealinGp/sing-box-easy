<script setup lang="ts">
import { computed, ref } from 'vue'
import * as monaco from 'monaco-editor'
import Button from '../components/Button.vue'
import Input from '../components/Input.vue'
import Textarea from '../components/Textarea.vue'
import Select from '../volt/Select.vue'
import Dialog from '../volt/Dialog.vue'
import MonacoEditor from '../components/MonacoEditor.vue'
import MonacoDiffEditor from '../components/MonacoDiffEditor.vue'

type Rule = Record<string, unknown>
interface Source { version: number; rules: Rule[] }
interface Saved { tag: string; path: string; content: Source }
const fields = [
  { key: 'domain', label: '完整域名', example: 'example.com' },
  { key: 'domain_suffix', label: '域名后缀', example: 'example.org' },
  { key: 'domain_keyword', label: '域名关键词', example: 'shopify' },
  { key: 'domain_regex', label: '域名正则', example: '^stun\\..+' },
  { key: 'ip_cidr', label: '目标 IP 网段', example: '192.168.1.0/24' },
  { key: 'source_ip_cidr', label: '来源 IP 网段', example: '10.0.0.0/8' },
  { key: 'port', label: '目标端口', example: '443', numeric: true },
  { key: 'port_range', label: '目标端口范围', example: '1000:2000' },
  { key: 'source_port', label: '来源端口', example: '8080', numeric: true },
  { key: 'source_port_range', label: '来源端口范围', example: '1000:2000' },
  { key: 'network', label: '网络协议', example: 'tcp' },
  { key: 'query_type', label: 'DNS 查询类型', example: 'A' },
  { key: 'process_name', label: '进程名称', example: 'curl' },
  { key: 'process_path', label: '进程路径', example: '/usr/bin/curl' },
  { key: 'process_path_regex', label: '进程路径正则', example: '^/usr/bin/.+' },
]
const properties = Object.fromEntries(fields.map(f => [f.key, {
  description: f.label, type: 'array', items: { type: f.numeric ? 'integer' : 'string' },
  default: [], examples: [[f.numeric ? 443 : f.example]],
}]))
monaco.json.jsonDefaults.setDiagnosticsOptions({
  validate: true,
  schemas: [{ uri: 'https://preview.local/rule-set.schema.json', fileMatch: ['*'],
    schema: { type: 'object', required: ['version', 'rules'], properties: {
      version: { type: 'integer', enum: [3, 4, 5] },
      rules: { type: 'array', items: { type: 'object', properties: {
        ...properties, invert: { type: 'boolean', description: '取反' },
        type: { enum: ['logical'], description: '逻辑组合' },
        mode: { enum: ['and', 'or'] }, rules: { type: 'array' },
      } } },
    } },
  }],
})
const saved = ref<Saved[]>([{
  tag: '我的代理域名', path: '/etc/sing-box/rule-sets/user/我的代理域名.json',
  content: { version: 3, rules: [{ domain_suffix: ['example.com', 'example.org'], domain_keyword: ['shopify'] }] },
}])
const editing = ref(false)
const selected = ref<number | null>(null)
const tag = ref('')
const source = ref<Source>({ version: 3, rules: [] })
const mode = ref('form')
const raw = ref('')
const error = ref('')
const notice = ref('')
const showDiff = ref(false)
const pending = ref<Source | null>(null)
const before = ref<Source>({ version: 3, rules: [] })
const nextField = ref<string | null>(null)
const filePath = computed(() => {
  const filename = tag.value.trim().replace(/[<>:"/\\|?*\u0000-\u001f]/g, '_').replace(/[. ]+$/g, '')
  return '/etc/sing-box/rule-sets/user/' + (filename || '未命名规则集') + '.json'
})
const serialize = (v: unknown) => JSON.stringify(v, null, 2)
function open(index: number | null) {
  selected.value = index
  const item = index === null ? null : saved.value[index]
  tag.value = item?.tag ?? ''
  source.value = JSON.parse(serialize(item?.content ?? { version: 3, rules: [{ domain: [], domain_suffix: [] }] }))
  before.value = JSON.parse(serialize(item?.content ?? { version: 3, rules: [] }))
  raw.value = serialize(source.value)
  mode.value = 'form'
  error.value = ''
  notice.value = ''
  editing.value = true
}
function changeMode(value: string) {
  error.value = ''
  if (value === 'json') raw.value = serialize(source.value)
  else {
    try {
      const parsed = JSON.parse(raw.value)
      if (!parsed || !Array.isArray(parsed.rules)) throw new Error('需要 version 和 rules 数组。')
      if (parsed.rules.some((r: Rule) => !r || Array.isArray(r) || typeof r !== 'object' ||
        Object.keys(r).some(key => !fields.some(f => f.key === key)))) {
        throw new Error('此草稿包含逻辑组合或高级字段，请继续在 JSON 中编辑。内容已保留。')
      }
      source.value = parsed
    } catch (e) { error.value = e instanceof Error ? e.message : String(e); return }
  }
  mode.value = value
}
function values(value: unknown) { return Array.isArray(value) ? value.join('\n') : String(value ?? '') }
function update(rule: Rule, key: string, value: string) {
  const parts = value.split(/\r?\n/).map(s => s.trim()).filter(Boolean)
  rule[key] = fields.find(f => f.key === key)?.numeric ? parts.map(Number) : parts
}
function preview() {
  error.value = ''
  try {
    if (!tag.value.trim()) throw new Error('请填写规则集名称。')
    if (saved.value.some((s, i) => i !== selected.value && s.tag === tag.value.trim())) throw new Error('规则集名称已存在。')
    if (saved.value.some((s, i) => i !== selected.value && s.path === filePath.value)) throw new Error('生成的文件路径已被其他规则集使用，请调整名称。')
    const data = mode.value === 'json' ? JSON.parse(raw.value) : JSON.parse(serialize(source.value))
    if (![3, 4, 5].includes(data.version) || !Array.isArray(data.rules)) throw new Error('需要 version（3–5）与 rules 数组。')
    data.rules = data.rules.map((r: Rule) => {
      if (!r || typeof r !== 'object' || Array.isArray(r)) throw new Error('每条规则必须是对象。')
      return Object.fromEntries(Object.entries(r).filter(([, v]) => !(Array.isArray(v) && !v.length)))
    })
    if (!data.rules.length || data.rules.some((r: Rule) => !Object.keys(r).length)) throw new Error('每条规则至少填写一个条件。')
    pending.value = data
    showDiff.value = true
  } catch (e) { error.value = e instanceof Error ? e.message : String(e) }
}
function save() {
  const item = { tag: tag.value.trim(), path: filePath.value, content: pending.value! }
  if (selected.value === null) saved.value.push(item)
  else saved.value[selected.value] = item
  showDiff.value = false
  editing.value = false
  notice.value = '已模拟保存 · 路径已自动填写，可以再次编辑规则内容。'
}
</script>

<template>
  <main class="mx-auto max-w-6xl space-y-6 px-4 py-8 text-text-primary">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div><p class="mb-2 text-xs text-text-secondary">交互预览 · 数据仅保存在本页内存，刷新还原</p>
        <h1 class="text-2xl font-semibold">本地规则集</h1></div>
      <Button @click="open(null)" :disabled="editing">创建本地规则集</Button>
    </div>
    <p v-if="notice" role="status" class="rounded-control bg-green-50 p-3 text-sm text-green-800">{{ notice }}</p>
    <section v-if="!editing" class="space-y-3">
      <article v-for="(item, index) in saved" :key="item.path" class="flex flex-wrap items-center justify-between gap-4 rounded-surface border border-border bg-bg-primary p-5">
        <div class="min-w-0 space-y-2"><h2 class="font-semibold">{{ item.tag }}</h2>
          <p class="break-all text-xs text-text-secondary">{{ item.path }}</p>
          <p class="text-sm text-text-secondary">{{ item.content.rules.length }} 条规则 · 本地 JSON</p>
        </div><Button variant="secondary" @click="open(index)">编辑规则内容</Button>
      </article>
    </section>
    <section v-else class="space-y-5 rounded-surface border border-border bg-bg-primary p-5">
      <div class="grid gap-5 md:grid-cols-2">
        <Input v-model="tag" label="规则集名称" placeholder="我的代理域名" />
        <div class="space-y-2"><p class="text-sm font-medium">保存路径（模拟）</p>
          <p class="break-all rounded-control bg-bg-secondary p-3 font-mono text-xs">{{ filePath }}</p></div>
      </div>
      <div class="flex items-center justify-between gap-3 border-b border-border pb-3">
        <div class="flex gap-2" role="group" aria-label="编辑方式">
          <Button v-for="option in [{ value: 'form', label: '可视化' }, { value: 'json', label: 'JSON' }]" :key="option.value"
            :variant="mode === option.value ? 'primary' : 'ghost'" :aria-pressed="mode === option.value" @click="changeMode(option.value)">{{ option.label }}</Button>
        </div><span class="text-xs text-text-secondary">source · version {{ source.version }}</span>
      </div>
      <template v-if="mode === 'form'">
        <article v-for="(rule, index) in source.rules" :key="index" class="space-y-4 rounded-control border border-border p-4">
          <div class="flex justify-between"><h3 class="text-sm font-semibold">匹配条件 {{ index + 1 }}</h3>
            <Button v-if="source.rules.length > 1" variant="ghost" @click="source.rules.splice(index, 1)">删除此规则</Button></div>
          <div class="grid gap-4 md:grid-cols-2">
            <div v-for="key in Object.keys(rule)" :key="key">
              <div class="mb-1 flex items-center justify-between"><span class="text-xs text-text-secondary">{{ key }}</span>
                <button type="button" class="px-2 text-xs text-text-secondary underline" @click="delete rule[key]">移除</button></div>
              <Textarea :model-value="values(rule[key])" @update:model-value="value => update(rule, key, String(value))"
                :label="fields.find(f => f.key === key)?.label ?? key" :placeholder="fields.find(f => f.key === key)?.example" :rows="4" />
            </div>
          </div>
          <Select :model-value="nextField" :options="fields.filter(f => !(f.key in rule))" option-label="label" option-value="key"
            filter placeholder="添加匹配条件" aria-label="添加匹配条件" class="w-full md:w-72"
            @update:model-value="(value: string) => { rule[value] = []; nextField = null }">
            <template #option="{ option }"><span>{{ option.label }} <span class="ml-2 text-xs text-text-secondary">{{ option.key }}</span></span></template>
          </Select>
        </article>
        <Button variant="ghost" @click="source.rules.push({ domain_suffix: [] })">＋ 添加另一条规则</Button>
        <p class="text-xs text-text-secondary">同一域名条件中的多项满足任意一项即可。添加端口或协议会进一步限定匹配范围。</p>
      </template>
      <div v-else class="h-[52vh] overflow-hidden rounded-control border border-border">
        <MonacoEditor v-model="raw" :options="{ minimap: { enabled: false } }" class="h-full" />
      </div>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
      <div class="flex justify-end gap-3 border-t border-border pt-4">
        <Button variant="ghost" @click="editing = false">取消</Button>
        <Button @click="preview">预览并保存</Button>
      </div>
    </section>
    <Dialog v-model:visible="showDiff" modal header="保存本地规则文件 · 差异对比" class="w-full max-w-6xl">
      <p class="mb-3 break-all text-xs text-text-secondary">{{ filePath }}</p>
      <div class="flex justify-between text-xs"><span>保存前</span><span>保存后</span></div>
      <div class="mt-2 h-[56vh]"><MonacoDiffEditor :original="serialize(before)" :modified="serialize(pending)" /></div>
      <template #footer><Button variant="ghost" @click="showDiff = false">返回编辑</Button><Button @click="save">模拟保存</Button></template>
    </Dialog>
  </main>
</template>
