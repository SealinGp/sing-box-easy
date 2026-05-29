<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue'
import * as monaco from 'monaco-editor'

interface Props {
  original: string
  modified: string
  language?: string
  theme?: string
}

const props = withDefaults(defineProps<Props>(), {
  language: 'json',
  theme: 'vs-dark',
})

const container = ref<HTMLDivElement>()
let diffEditor: monaco.editor.IStandaloneDiffEditor | null = null
let originalModel: monaco.editor.ITextModel | null = null
let modifiedModel: monaco.editor.ITextModel | null = null

// Monaco worker bootstrap lives in `src/plugins/monaco.ts` (runs once at
// startup). Do NOT reassign self.MonacoEnvironment here.

const buildModels = () => {
  originalModel = monaco.editor.createModel(props.original, props.language)
  modifiedModel = monaco.editor.createModel(props.modified, props.language)
  diffEditor?.setModel({ original: originalModel, modified: modifiedModel })
}

const disposeModels = () => {
  originalModel?.dispose()
  modifiedModel?.dispose()
  originalModel = null
  modifiedModel = null
}

onMounted(() => {
  if (!container.value) return
  diffEditor = monaco.editor.createDiffEditor(container.value, {
    theme: props.theme,
    readOnly: true,
    automaticLayout: true,
    renderSideBySide: true,
    minimap: { enabled: false },
    fontSize: 13,
    scrollBeyondLastLine: false,
  })
  buildModels()
})

// Rebuild models whenever either side changes (e.g. user picks another version).
watch(
  () => [props.original, props.modified],
  () => {
    disposeModels()
    buildModels()
  }
)

watch(
  () => props.theme,
  (t) => monaco.editor.setTheme(t)
)

onBeforeUnmount(() => {
  disposeModels()
  diffEditor?.dispose()
})
</script>

<template>
  <div ref="container" class="w-full h-full min-h-[300px]"></div>
</template>
