import { defineComponent, h, cloneVNode, TransitionGroup, Teleport, computed, ref, watch, onBeforeUnmount, type VNode } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { ClipboardDocumentIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import CardActions from './CardActions.vue'
import CollectionCell from './CollectionCell.vue'
import CollectionViewToggle from './CollectionViewToggle.vue'
import { elements, nodeText, containsCheckbox } from '../utils/tableVNodes'
import { useCollectionToolbarTarget } from '../composables/useCollectionToolbarTarget'
import { useTableColumns } from '../composables/useTableColumns'
import type { TableColumn } from './Table.vue'

export default defineComponent({
  name: 'TableRecords',
  props: {
    columns: { type: Array as () => TableColumn[], default: () => [] },
    tableClass: String,
    storageKey: String,
    presentation: { type: Boolean, default: true },
    loading: Boolean,
    transition: Boolean,
    selectable: { type: Boolean, default: true },
  },
  setup(props, { slots }) {
    const { t } = useI18n()
    const route = useRoute()
    const root = ref<HTMLElement | null>(null)
    const toolbarTarget = useCollectionToolbarTarget(root)
    const mode = ref<'list' | 'cards'>('list')
    const selected = ref(new Set<string>())
    const error = ref('')
    const copied = ref(false)
    let timer: ReturnType<typeof setTimeout> | undefined
    const key = computed(() => props.storageKey || `collection:${route.path}`)
    watch(key, value => {
      selected.value = new Set()
      try { mode.value = localStorage.getItem(`${value}:view`) === 'cards' ? 'cards' : 'list' } catch {}
    }, { immediate: true })
    function changeMode(value: 'list' | 'cards') {
      mode.value = value
      try { localStorage.setItem(`${key.value}:view`, value) } catch {}
    }
    useTableColumns(root, () => mode.value === 'list' && !props.loading, () => `${key.value}:columns`)
    onBeforeUnmount(() => clearTimeout(timer))
    function checkbox(checked: boolean, label: string, change: () => void, partial = false) {
      return h('input', {
        type: 'checkbox', checked, indeterminate: partial, 'aria-label': label,
        class: 'collection-checkbox', onChange: change,
        onClick: (event: Event) => event.stopPropagation(),
        onMousedown: (event: Event) => event.stopPropagation(),
      })
    }
    return () => {
      const rows = elements(slots.default?.(), 'tr')
      const headers = slots.head ? elements(slots.head(), 'th') : props.columns.map(column =>
        h('th', { key: column.key, class: [column.class, column.align === 'right' ? 'col-actions' : ''], 'data-min-width': column.minWidth }, column.label))
      // Preserve pages that already own selection and their business bulk actions.
      const selectionHeader = headers.find(header => String(header.props?.class).includes('col-select') || containsCheckbox(header.children))
      const ownedSelection = !!selectionHeader
      const selectable = props.selectable && !ownedSelection && !props.loading
      const records = rows.filter(row => !elements(row.children, 'td').some(cell => Number(cell.props?.colspan) > 1))
      const identity = (row: VNode) => String(row.key ?? rows.indexOf(row))
      const visible = records.map(identity)
      const checked = visible.filter(id => selected.value.has(id))
      const toggle = (id: string) => {
        const next = new Set(selected.value)
        next.has(id) ? next.delete(id) : next.add(id)
        selected.value = next
      }
      const selectAll = () => {
        const next = new Set(selected.value)
        visible.forEach(id => checked.length === visible.length ? next.delete(id) : next.add(id))
        selected.value = next
      }
      const copy = async () => {
        error.value = ''
        const text = records.filter(row => selected.value.has(identity(row))).map(row =>
          elements(row.children, 'td').map(cell => nodeText(cell.children)).join('\t')).join('\n')
        try {
          // DOM text includes component-rendered status labels; never copy hidden data.
          const activeRows = [...(root.value?.querySelectorAll<HTMLElement>('[data-collection-selected="true"]') ?? [])]
          await navigator.clipboard.writeText(activeRows.length ? activeRows.map(row => row.innerText).join('\n') : text)
          copied.value = true
          clearTimeout(timer)
          timer = setTimeout(() => { copied.value = false }, 1800)
        } catch { error.value = t('common.copyFailed') }
      }
      const toolbar = props.presentation || selectable ? h('div', { class: 'collection-toolbar' }, [
        selectable && mode.value === 'cards' ? checkbox(checked.length === visible.length && !!visible.length, t('common.collection.selectAll'), selectAll, !!checked.length && checked.length < visible.length) : null,
        ownedSelection && mode.value === 'cards' ? h('span', { class: 'collection-owned-selection' }, selectionHeader?.children as VNode[]) : null,
        selectable && checked.length ? h('div', { class: 'collection-selection' }, [
          h('span', { role: 'status' }, t('common.collection.selected', { count: checked.length })),
          h('button', { type: 'button', class: 'collection-tool', title: t('common.copy'), 'aria-label': t('common.copy'), onClick: copy }, [h(ClipboardDocumentIcon), copied.value ? t('common.copied') : null]),
          h('button', { type: 'button', class: 'collection-tool', title: t('common.collection.clear'), 'aria-label': t('common.collection.clear'), onClick: () => { selected.value = new Set() } }, [h(XMarkIcon)]),
        ]) : null,
        props.presentation ? h(CollectionViewToggle, { modelValue: mode.value, 'onUpdate:modelValue': changeMode }) : null,
      ]) : null
      const labels = headers.map(header => nodeText(header.children))
      const showCards = props.presentation && mode.value === 'cards' && !props.loading
      const tableRows = rows.map(row => {
          const cells = elements(row.children, 'td')
          const detail = cells.some(cell => Number(cell.props?.colspan) > 1)
          return h('tr', { ...row.props, key: row.key ?? undefined, 'data-collection-selected': selected.value.has(identity(row)) }, [
            selectable && !detail ? h('td', { class: 'col-select' }, [checkbox(selected.value.has(identity(row)), t('common.collection.select'), () => toggle(identity(row)))]) : null,
            ...cells.map(cell => detail ? cloneVNode(cell, { colspan: Number(cell.props?.colspan) + (selectable ? 1 : 0) }) : h('td', cell.props ?? {}, [h(CollectionCell, {}, { default: () => cell.children })])),
          ])
        })
      const content = showCards && slots.cards ? slots.cards({ selected: selected.value, toggle }) : showCards ? h('div', { class: 'collection-cards', role: 'list' }, rows.map(row => {
        const cells = elements(row.children, 'td')
        return h('article', {
          ...row.props, key: row.key ?? undefined, role: 'listitem', class: ['collection-card', row.props?.class],
          'data-collection-selected': selected.value.has(identity(row)),
        }, [
          selectable && records.includes(row) ? checkbox(selected.value.has(identity(row)), t('common.collection.select'), () => toggle(identity(row))) : null,
          ...cells.map((cell, index) => String(cell.props?.class).includes('col-actions') ? h(CardActions, {}, { default: () => cell.children }) : h('section', { class: ['collection-card-field', cell.props?.class] }, [
            labels[index] ? h('div', { class: 'collection-field-label' }, labels[index]) : null,
            h('div', { class: 'collection-field-content' }, cell.children as VNode[]),
          ])),
        ])
      })) : h('table', { class: ['data-table', props.tableClass, { 'collection-compact': props.presentation }] }, [
        h('thead', [h('tr', [
          selectable ? h('th', { class: 'col-select', 'data-min-width': 56 }, [checkbox(checked.length === visible.length && !!visible.length, t('common.collection.selectAll'), selectAll, !!checked.length && checked.length < visible.length)]) : null,
          ...headers,
        ])]),
        props.transition ? h(TransitionGroup, { tag: 'tbody', moveClass: 'ui-row-move' }, () => tableRows) : h('tbody', tableRows),
      ])
      return h('div', { ref: root, class: 'collection-records' }, [toolbar && toolbarTarget.value ? h(Teleport, { to: toolbarTarget.value }, [toolbar]) : toolbar, error.value ? h('p', { role: 'alert' }, error.value) : null,
        h('div', { key: showCards ? 'cards' : 'list', class: 'collection-view' }, [content])])
    }
  },
})
