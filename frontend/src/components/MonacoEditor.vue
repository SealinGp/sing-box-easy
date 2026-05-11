<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue'
import * as monaco from 'monaco-editor'

interface Props {
  modelValue: string
  language?: string
  theme?: string
  options?: any
}

const props = withDefaults(defineProps<Props>(), {
  language: 'json',
  theme: 'vs-dark',
  options: () => ({})
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorContainer = ref<HTMLDivElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null

// Monaco worker bootstrap lives in `src/plugins/monaco.ts` and runs once at
// application startup. Do NOT re-assign `self.MonacoEnvironment` here — that
// would clobber the global config every time a MonacoEditor instance mounts.

onMounted(() => {
  if (editorContainer.value) {
    // Create the editor
    editor = monaco.editor.create(editorContainer.value, {
      value: props.modelValue,
      language: props.language,
      theme: props.theme,
      automaticLayout: true,
      minimap: { enabled: true },
      scrollBeyondLastLine: false,
      fontSize: 14,
      tabSize: 2,
      formatOnType: true,
      formatOnPaste: true,
      folding: true,
      showFoldingControls: 'always',
      foldingStrategy: 'auto',
      foldingHighlight: true,
      lineNumbers: 'on',
      renderLineHighlight: 'all',
      bracketPairColorization: {
        enabled: true,
      },
      wordWrap: 'on',
      ...props.options
    })

    // Listen for content changes
    editor.onDidChangeModelContent(() => {
      const value = editor?.getValue()
      if (value !== undefined) {
        emit('update:modelValue', value)
      }
    })
  }
})

// Watch for external value changes
watch(() => props.modelValue, (newValue) => {
  if (editor && editor.getValue() !== newValue) {
    editor.setValue(newValue)
  }
})

// Watch for theme changes
watch(() => props.theme, (newTheme) => {
  if (editor) {
    monaco.editor.setTheme(newTheme)
  }
})

// Watch for options changes
watch(() => props.options, (newOptions) => {
  if (editor) {
    editor.updateOptions(newOptions)
  }
}, { deep: true })

// Watch for language changes so a parent can flip e.g. json -> yaml at runtime.
watch(() => props.language, (lang) => {
  const model = editor?.getModel()
  if (model && lang) {
    monaco.editor.setModelLanguage(model, lang)
  }
})

onBeforeUnmount(() => {
  editor?.dispose()
})
</script>

<template>
  <div ref="editorContainer" class="w-full h-full min-h-[400px]"></div>
</template>