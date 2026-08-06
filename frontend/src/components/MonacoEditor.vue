<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, nextTick } from 'vue'
import * as monaco from 'monaco-editor'

interface Props {
  modelValue: string
  language?: string
  theme?: string
  options?: any
  /**
   * When true, renders a second editor pane to the right that shares the same
   * underlying text model. Both panes stay in sync (edit one, see it in the
   * other) while scrolling/folding independently — a true side-by-side view.
   */
  split?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  language: 'json',
  theme: 'vs-dark',
  options: () => ({}),
  split: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const wrapper = ref<HTMLDivElement>()
const primaryContainer = ref<HTMLDivElement>()
const secondaryContainer = ref<HTMLDivElement>()

// A single shared model backs every pane so their contents never diverge.
let model: monaco.editor.ITextModel | null = null
let primaryEditor: monaco.editor.IStandaloneCodeEditor | null = null
let secondaryEditor: monaco.editor.IStandaloneCodeEditor | null = null

// Build the common construction options once so both panes look identical.
const baseOptions = (): monaco.editor.IStandaloneEditorConstructionOptions => ({
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
  ...props.options,
})

onMounted(() => {
  // Create the shared model explicitly (instead of passing `value:`) so we can
  // attach it to multiple editors.
  model = monaco.editor.createModel(props.modelValue, props.language)

  if (primaryContainer.value) {
    primaryEditor = monaco.editor.create(primaryContainer.value, {
      model,
      ...baseOptions(),
    })
  }

  // One listener on the shared model covers edits from any pane.
  model.onDidChangeContent(() => {
    const value = model?.getValue()
    if (value !== undefined) {
      emit('update:modelValue', value)
    }
  })

  if (props.split) {
    createSecondaryEditor()
  }
})

const createSecondaryEditor = async () => {
  // Wait for v-show to reveal the container so Monaco can measure it.
  await nextTick()
  if (secondaryContainer.value && model && !secondaryEditor) {
    secondaryEditor = monaco.editor.create(secondaryContainer.value, {
      model,
      ...baseOptions(),
    })
  }
}

const disposeSecondaryEditor = () => {
  // Dispose only the editor — the shared model is owned here and disposed on
  // unmount, never when a pane closes.
  secondaryEditor?.dispose()
  secondaryEditor = null
}

// ---- Draggable splitter -------------------------------------------------
const splitRatio = ref(0.5)
let dragging = false

const relayout = () => {
  primaryEditor?.layout()
  secondaryEditor?.layout()
}

const onDividerMouseDown = (e: MouseEvent) => {
  e.preventDefault()
  dragging = true
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

const onMouseMove = (e: MouseEvent) => {
  if (!dragging || !wrapper.value) return
  const rect = wrapper.value.getBoundingClientRect()
  const ratio = (e.clientX - rect.left) / rect.width
  // Clamp so neither pane collapses to an unusable sliver.
  splitRatio.value = Math.min(0.8, Math.max(0.2, ratio))
  relayout()
}

const onMouseUp = () => {
  if (!dragging) return
  dragging = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
}

// ---- Reactive props -----------------------------------------------------
watch(() => props.split, (val) => {
  if (val) {
    createSecondaryEditor()
  } else {
    disposeSecondaryEditor()
  }
  // Primary pane width changes either way; relayout after the DOM settles.
  nextTick(relayout)
})

watch(() => props.modelValue, (newValue) => {
  if (model && model.getValue() !== newValue) {
    model.setValue(newValue)
  }
})

watch(() => props.theme, (newTheme) => {
  monaco.editor.setTheme(newTheme)
})

watch(() => props.options, (newOptions) => {
  primaryEditor?.updateOptions(newOptions)
  secondaryEditor?.updateOptions(newOptions)
}, { deep: true })

watch(() => props.language, (lang) => {
  if (model && lang) {
    monaco.editor.setModelLanguage(model, lang)
  }
})

onBeforeUnmount(() => {
  onMouseUp()
  primaryEditor?.dispose()
  secondaryEditor?.dispose()
  model?.dispose()
})
</script>

<template>
  <div ref="wrapper" class="flex w-full h-full min-h-[400px]">
    <div
      ref="primaryContainer"
      class="h-full min-h-[400px]"
      :style="split ? { width: splitRatio * 100 + '%' } : { width: '100%' }"
    ></div>

    <div
      v-show="split"
      class="w-1 shrink-0 cursor-col-resize bg-gray-300 dark:bg-gray-700 hover:bg-primary-500 transition-colors"
      :title="$t('config.dragResize')"
      @mousedown="onDividerMouseDown"
    ></div>

    <div
      v-show="split"
      ref="secondaryContainer"
      class="h-full min-h-[400px] flex-1"
    ></div>
  </div>
</template>
